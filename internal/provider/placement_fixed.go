package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithImportState = &PlacementResourceFixed{}
var _ resource.ResourceWithModifyPlan = &PlacementResourceFixed{}

type PlacementResourceFixed struct{ client *PveClient }

type PlacementResourceFixedModel struct {
	ID          types.String  `tfsdk:"id"`
	NodeName    types.String  `tfsdk:"node_name"`
	MemUsagePct types.Float64 `tfsdk:"memory_usage_pct"`
	CpuUsagePct types.Float64 `tfsdk:"cpu_usage_pct"`
	Zone        types.String  `tfsdk:"zone"`
}

func NewPlacementResourceFixed() resource.Resource {
	return &PlacementResourceFixed{}
}

func (r *PlacementResourceFixed) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_placement_fixed"
}

func (r *PlacementResourceFixed) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Pins VM placement to one explicitly named Proxmox node. The node must exist in the cluster when the resource is created. Changing `node_name` replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_name": schema.StringAttribute{
				Required:    true,
				Description: "The Proxmox node to place the VM on. Changing this forces a new placement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"memory_usage_pct": schema.Float64Attribute{
				Computed:    true,
				Description: "Memory utilization of the node at time of placement, as a percentage (0-100).",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"cpu_usage_pct": schema.Float64Attribute{
				Computed:    true,
				Description: "CPU utilization of the node at time of placement, as a percentage (0-100).",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"zone": schema.StringAttribute{
				Computed:    true,
				Description: "The zone of `node_name` in the provider topology at time of placement, or null if the node is in no zone or in more than one.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PlacementResourceFixed) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*PveClient)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *PlacementResourceFixed) lookupNode(name string, diags *diag.Diagnostics) *PveNode {
	pveNodes, err := r.client.GetNodes()
	if err != nil {
		diags.AddError("failed to fetch PVE nodes", err.Error())
		return nil
	}
	for i := range pveNodes {
		if pveNodes[i].Node == name {
			return &pveNodes[i]
		}
	}
	diags.AddError("node not found", fmt.Sprintf("no PVE node named %q is reachable through this provider", name))
	return nil
}

func fixedPlacementState(n *PveNode, topology map[string][]string) PlacementResourceFixedModel {
	state := PlacementResourceFixedModel{
		ID:          types.StringValue(n.Node),
		NodeName:    types.StringValue(n.Node),
		MemUsagePct: types.Float64Null(),
		CpuUsagePct: types.Float64Value(n.Cpu * 100),
		Zone:        zoneOf(topology, n.Node),
	}
	if n.MaxMem > 0 {
		state.MemUsagePct = types.Float64Value((n.Mem / n.MaxMem) * 100)
	}
	return state
}

func (r *PlacementResourceFixed) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PlacementResourceFixedModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	node := r.lookupNode(plan.NodeName.ValueString(), &resp.Diagnostics)
	if node == nil {
		return
	}

	state := fixedPlacementState(node, r.client.Topology)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read is intentionally a no-op: the placement decision is locked in state.
func (r *PlacementResourceFixed) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PlacementResourceFixedModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlacementResourceFixed) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"placement cannot be updated in place",
		"node_name of pvescheduler_placement_fixed requires replacement, so this code path should be unreachable. Please report this as a provider bug.",
	)
}

func (r *PlacementResourceFixed) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	nodeName := strings.TrimSpace(req.ID)
	if nodeName == "" {
		resp.Diagnostics.AddError(
			"invalid import ID",
			"Expected a PVE node name, for example: terraform import pvescheduler_placement_fixed.vm pve01",
		)
		return
	}

	node := r.lookupNode(nodeName, &resp.Diagnostics)
	if node == nil {
		return
	}

	state := fixedPlacementState(node, r.client.Topology)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlacementResourceFixed) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// no-op: placement is a logical decision with no PVE-side resource to clean up
}

func (r *PlacementResourceFixed) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !req.State.Raw.IsNull() || r.client == nil {
		return
	}

	var nodeName types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("node_name"), &nodeName)...)
	if resp.Diagnostics.HasError() || nodeName.IsNull() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("zone"), zoneOf(r.client.Topology, nodeName.ValueString()))...)
}
