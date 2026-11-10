package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func testAccMoveFromPlacement(t *testing.T, address, config string, action plancheck.ResourceActionType, checks ...resource.TestCheckFunc) resource.TestCase {
	t.Helper()
	endpoint := testAccStubPVE(t)
	return resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + `
resource "pvescheduler_placement" "vm" {}
`,
				Check: resource.TestCheckResourceAttr("pvescheduler_placement.vm", "node_name", "pve02"),
			},
			{
				Config: testAccProviderBlock(endpoint, testAccTopology) + config + fmt.Sprintf(`
moved {
  from = pvescheduler_placement.vm
  to   = %s
}
`, address),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, action),
					},
				},
				Check: resource.ComposeTestCheckFunc(checks...),
			},
		},
	}
}

func TestAccPlacementDynamic_MoveFromPlacement(t *testing.T) {
	resource.Test(t, testAccMoveFromPlacement(t, "pvescheduler_placement_dynamic.vm", `
resource "pvescheduler_placement_dynamic" "vm" {
  zone = "b"
}
`, plancheck.ResourceActionNoop,
		resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "node_name", "pve02"),
		resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "zone", "b"),
	))
}

func TestAccPlacementDynamic_MoveFromPlacementOtherZoneReplaces(t *testing.T) {
	resource.Test(t, testAccMoveFromPlacement(t, "pvescheduler_placement_dynamic.vm", `
resource "pvescheduler_placement_dynamic" "vm" {
  zone = "a"
}
`, plancheck.ResourceActionReplace,
		resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "node_name", "pve03"),
		resource.TestCheckResourceAttr("pvescheduler_placement_dynamic.vm", "zone", "a"),
	))
}

func TestAccPlacementFixed_MoveFromPlacement(t *testing.T) {
	resource.Test(t, testAccMoveFromPlacement(t, "pvescheduler_placement_fixed.vm", `
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve02"
}
`, plancheck.ResourceActionNoop,
		resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "node_name", "pve02"),
		resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "zone", "b"),
	))
}

func TestAccPlacementFixed_MoveFromPlacementOtherNodeReplaces(t *testing.T) {
	resource.Test(t, testAccMoveFromPlacement(t, "pvescheduler_placement_fixed.vm", `
resource "pvescheduler_placement_fixed" "vm" {
  node_name = "pve01"
}
`, plancheck.ResourceActionReplace,
		resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "node_name", "pve01"),
		resource.TestCheckResourceAttr("pvescheduler_placement_fixed.vm", "zone", "a"),
	))
}
