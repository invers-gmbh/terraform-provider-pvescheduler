package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"pvescheduler": providerserver.NewProtocol6WithError(New()),
}

var (
	regexpUnsupportedArgument   = regexp.MustCompile(`Unsupported argument`)
	regexpNodeNotFound          = regexp.MustCompile(`node not found`)
	regexpInvalidAttributeValue = regexp.MustCompile(`Invalid Attribute Value`)
	regexpBuildAllowlist        = regexp.MustCompile(`failed to build node allowlist`)
	// Wrapped across lines by Terraform's diagnostic renderer, so match one word.
	regexpCertificate = regexp.MustCompile(`certificate`)
)

// Utilisation values are chosen to be exactly representable in binary so the
// generated percentages are stable strings: 0.25, 0.5 and 0.75.
func testAccPVEHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []PveNode{
				{Node: "pve01", Status: "online", Mem: 96, MaxMem: 128, Cpu: 0.75},
				{Node: "pve02", Status: "online", Mem: 32, MaxMem: 128, Cpu: 0.25},
				{Node: "pve03", Status: "online", Mem: 64, MaxMem: 128, Cpu: 0.50},
			},
		})
	})
}

func testAccStubPVE(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(testAccPVEHandler())
	t.Cleanup(srv.Close)
	return srv.URL
}

// Served with a self-signed certificate, so a client that verifies fails and a
// client that skips verification succeeds.
func testAccStubPVETLS(t *testing.T) string {
	t.Helper()
	srv := httptest.NewTLSServer(testAccPVEHandler())
	t.Cleanup(srv.Close)
	return srv.URL
}

func testAccProviderBlock(endpoint, extra string) string {
	return fmt.Sprintf(`
provider "pvescheduler" {
  endpoint  = %q
  api_token = "root@pam!test=secret"
  %s
}
`, endpoint, extra)
}

// Scores are 0.75, 0.25 and 0.50, so pve02 wins on the default weights.
func TestAccPlacement_PicksLowestScoringNode(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pvescheduler_placement.vm", "node_name", "pve02"),
					resource.TestCheckResourceAttr("pvescheduler_placement.vm", "id", "pve02"),
					resource.TestCheckResourceAttr("pvescheduler_placement.vm", "memory_usage_pct", "25"),
					resource.TestCheckResourceAttr("pvescheduler_placement.vm", "cpu_usage_pct", "25"),
				),
			},
		},
	})
}

// The weights carry schema defaults, so they must appear in state as concrete
// values rather than null when the configuration omits them.
func TestAccPlacement_WeightDefaultsAppearInState(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pvescheduler_placement.vm", "memory_weight", "0.7"),
					resource.TestCheckResourceAttr("pvescheduler_placement.vm", "cpu_weight", "0.3"),
				),
			},
		},
	})
}

func TestAccPlacement_RejectsNegativeWeight(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {
  memory_weight = -1
}
`,
				ExpectError: regexpInvalidAttributeValue,
			},
		},
	})
}

func TestAccNodeDataSource_RejectsNegativeWeight(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
data "pvescheduler_node" "best" {
  cpu_weight = -0.5
}
`,
				ExpectError: regexpInvalidAttributeValue,
			},
		},
	})
}

func TestAccPlacement_RespectsAllowlistAndExclude(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, `nodes = ["pve01", "pve03"]`) + `
resource "pvescheduler_placement" "vm" {
  exclude = ["pve01"]
}
`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement.vm", "node_name", "pve03"),
			},
		},
	})
}

// Regression test for the silent-success defect: adding exclude must replace the
// resource and actually move the VM, not report success while leaving it put.
func TestAccPlacement_ExcludeForcesReplacement(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {}
`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement.vm", "node_name", "pve02"),
			},
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {
  exclude = ["pve02"]
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("pvescheduler_placement.vm", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("pvescheduler_placement.vm", "node_name", "pve03"),
			},
		},
	})
}

func TestAccPlacement_WeightChangeForcesReplacement(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {
  memory_weight = 0.7
  cpu_weight    = 0.3
}
`,
			},
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {
  memory_weight = 0.9
  cpu_weight    = 0.1
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("pvescheduler_placement.vm", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

// Regression test for the published example that did not parse: nodes is a
// provider-level argument and must be rejected on the resource.
func TestAccPlacement_RejectsUnknownAttribute(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {
  nodes = ["pve01"]
}
`,
				ExpectError: regexpUnsupportedArgument,
			},
		},
	})
}

func TestAccPlacement_Import(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {}
`,
			},
			{
				ResourceName:      "pvescheduler_placement.vm",
				ImportState:       true,
				ImportStateId:     "pve02",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlacement_ImportUnknownNodeFails(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement" "vm" {}
`,
			},
			{
				ResourceName:  "pvescheduler_placement.vm",
				ImportState:   true,
				ImportStateId: "pve99",
				ExpectError:   regexpNodeNotFound,
			},
		},
	})
}

func TestAccNodeDataSource_ReturnsLeastLoaded(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
data "pvescheduler_node" "best" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pvescheduler_node.best", "node_name", "pve02"),
					resource.TestCheckResourceAttr("data.pvescheduler_node.best", "memory_usage_pct", "25"),
				),
			},
		},
	})
}

func TestAccNodeDataSource_RejectsUnknownAttribute(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
data "pvescheduler_node" "best" {
  nodes = ["pve01"]
}
`,
				ExpectError: regexpUnsupportedArgument,
			},
		},
	})
}

// PROXMOX_VE_INSECURE must not override a value the practitioner set. Before the
// fix, ValueBool() returned false for both "unset" and "explicitly false", so an
// explicit false was silently downgraded and TLS verification was disabled.
func TestAccProvider_ExplicitTLSVerificationBeatsEnvironment(t *testing.T) {
	endpoint := testAccStubPVETLS(t)
	t.Setenv("PROXMOX_VE_INSECURE", "true")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, `insecure_skip_verify = false`) + `
data "pvescheduler_node" "best" {}
`,
				ExpectError: regexpCertificate,
			},
		},
	})
}

func TestAccProvider_ExplicitInsecureSkipsVerification(t *testing.T) {
	endpoint := testAccStubPVETLS(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, `insecure_skip_verify = true`) + `
data "pvescheduler_node" "best" {}
`,
				Check: resource.TestCheckResourceAttr("data.pvescheduler_node.best", "node_name", "pve02"),
			},
		},
	})
}

// The environment is still consulted when the argument is absent.
func TestAccProvider_EnvironmentEnablesInsecureWhenUnset(t *testing.T) {
	endpoint := testAccStubPVETLS(t)
	t.Setenv("PROXMOX_VE_INSECURE", "true")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
data "pvescheduler_node" "best" {}
`,
				Check: resource.TestCheckResourceAttr("data.pvescheduler_node.best", "node_name", "pve02"),
			},
		},
	})
}

const testAccTopology = `topology = {
    a = ["pve01", "pve03"]
    b = ["pve02"]
  }`

// fixed

func TestAccPlacementFixed_PinsNode(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve01"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "node_name", "pve01"),
					resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "memory_usage_pct", "75"),
					resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "cpu_usage_pct", "75"),
				),
			},
			{
				ResourceName:      "pvescheduler_placement_fixed.vm",
				ImportState:       true,
				ImportStateId:     "pve01",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlacementFixed_UnknownNodeFails(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve99"
}
`,
				ExpectError: regexpNodeNotFound,
			},
		},
	})
}

func TestAccPlacementFixed_NodeChangeForcesReplacement(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve01"
}
`,
			},
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve03"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("pvescheduler_placement_fixed.vm", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "node_name", "pve03"),
			},
		},
	})
}

func TestAccPlacementFixed_ExportsZone(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_fixed" "placement" {
  node_name = "pve01"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("pvescheduler_placement_fixed.placement", tfjsonpath.New("zone"), knownvalue.StringExact("a")),
					},
				},
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_fixed.placement", "zone", "a"),
			},
			{
				ResourceName:      "pvescheduler_placement_fixed.placement",
				ImportState:       true,
				ImportStateId:     "pve01",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlacementFixed_ZoneNullWithoutTopology(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement_fixed" "placement" {
  node_name = "pve01"
}
`,
				Check: resource.TestCheckNoResourceAttr("pvescheduler_placement_fixed.placement", "zone"),
			},
		},
	})
}

func TestAccPlacementFixed_NodeChangeUpdatesZoneAtPlan(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_fixed" "placement" {
	node_name = "pve01"
}
`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_fixed.placement", "zone", "a"),
			},
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_fixed" "placement" {
  node_name = "pve02"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("pvescheduler_placement_fixed.placement", plancheck.ResourceActionReplace),
						plancheck.ExpectKnownValue("pvescheduler_placement_fixed.placement", tfjsonpath.New("zone"), knownvalue.StringExact("b")),
					},
				},
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_fixed.placement", "zone", "b"),
			},
		},
	})
}

// dynamic

func TestAccPlacmentDynamic_RespectsZone(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_dynamic" "vm" {
	zone = "a"
}
`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "node_name", "pve03"),
			},
		},
	})
}

func TestAccPlacementDynamic_UnknownZoneFails(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
						resource "pvescheduler_placement_dynamic" "vm" {
							zone = "z"
						}
					`,
				ExpectError: regexpBuildAllowlist,
			},
		},
	})
}

func TestAccPlacementDynamic_ZoneChangeForcesReplacement(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
						resource "pvescheduler_placement_dynamic" "vm" {
							zone = "a"
						}
					`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "node_name", "pve03"),
			},
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
						resource "pvescheduler_placement_dynamic" "vm" {
							zone = "b"
						}
					`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("pvescheduler_placement_dynamic.vm", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "node_name", "pve02"),
			},
		},
	})
}

func TestAccPlacementDynamic_ComputesZoneWhenUnset(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_dynamic" "placement" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.placement", "node_name", "pve02"),
					resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.placement", "zone", "b"),
				),
			},
			{
				ResourceName:      "pvescheduler_placement_dynamic.placement",
				ImportState:       true,
				ImportStateId:     "pve02",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlacementDynamic_ZoneNullWithoutTopology(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, "") + `
resource "pvescheduler_placement_dynamic" "placement" {}
`,
				Check: resource.TestCheckNoResourceAttr("pvescheduler_placement_dynamic.placement", "zone"),
			},
		},
	})
}

func TestAccPlacementDynamic_RemovingZoneKeepsPlacement(t *testing.T) {
	endpoint := testAccStubPVE(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_dynamic" "placement" {
  zone = "a"
}
`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.placement", "node_name", "pve03"),
			},
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement_dynamic" "placement" {}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("pvescheduler_placement_dynamic.placement", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.placement", "node_name", "pve03"),
					resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.placement", "zone", "a"),
				),
			},
		},
	})
}
