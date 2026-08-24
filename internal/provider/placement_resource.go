package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlacementResource struct{ client *PveClient }

type PlacementResourceModel struct {
	ID          types.String  `tfsdk:"id"`
	NodeName    types.String  `tfsdk:"node_name"`
	Exclude     types.List    `tfsdk:"exclude"`
	MemWeight   types.Float64 `tfsdk:"memory_weight"`
	CpuWeight   types.Float64 `tfsdk:"cpu_weight"`
	MemUsagePct types.Number  `tfsdk:"memory_usage_pct"`
	CpuUsagePct types.Number  `tfsdk:"cpu_usage_pct"`
}

func NewPlacementResource() resource.Resource {
	return &PlacementResource{}
}

func (r *PlacementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_placement"
}

func (r *PlacementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Selects and locks a Proxmox node for VM placement. The chosen node is stored in state and not re-evaluated on subsequent applies. Changing `exclude`, `memory_weight` or `cpu_weight` replaces the resource and re-runs selection. Remove from state to force re-scheduling without a configuration change.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_name": schema.StringAttribute{
				Computed:    true,
				Description: "The selected Proxmox node name, locked in state after initial placement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"exclude": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Node names to exclude from placement. Changing this forces a new placement.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"memory_weight": schema.Float64Attribute{
				Optional:    true,
				Description: "Weight applied to memory utilization when scoring nodes (default 0.7). Changing this forces a new placement.",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
			},
			"cpu_weight": schema.Float64Attribute{
				Optional:    true,
				Description: "Weight applied to CPU utilization when scoring nodes (default 0.3). Changing this forces a new placement.",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.RequiresReplace(),
				},
			},
			"memory_usage_pct": schema.NumberAttribute{
				Computed:    true,
				Description: "Memory utilization of the selected node at time of placement.",
				PlanModifiers: []planmodifier.Number{
					numberplanmodifier.UseStateForUnknown(),
				},
			},
			"cpu_usage_pct": schema.NumberAttribute{
				Computed:    true,
				Description: "CPU utilization of the selected node at time of placement.",
				PlanModifiers: []planmodifier.Number{
					numberplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PlacementResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PlacementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PlacementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pveNodes, err := r.client.GetNodes()
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch PVE nodes", err.Error())
		return
	}

	allowlist := toSet(r.client.Nodes)
	exclude := buildExcludeSet(ctx, plan.Exclude, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	memWeight, cpuWeight := resolveWeights(plan.MemWeight, plan.CpuWeight)

	best, err := selectNode(pveNodes, allowlist, exclude, memWeight, cpuWeight)
	if err != nil {
		resp.Diagnostics.AddError("node selection failed", err.Error())
		return
	}

	plan.ID = types.StringValue(best.Node)
	plan.NodeName = types.StringValue(best.Node)
	plan.MemUsagePct = types.NumberValue(bigFloat((best.Mem / best.MaxMem) * 100))
	plan.CpuUsagePct = types.NumberValue(bigFloat(best.Cpu * 100))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is intentionally a no-op: the placement decision is locked in state.
// Remove the resource from state (terraform state rm) to force re-scheduling.
func (r *PlacementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PlacementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlacementResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"placement cannot be updated in place",
		"Every configurable attribute of pvescheduler_placement requires replacement, so this code path should be unreachable. Please report this as a provider bug.",
	)
}

func (r *PlacementResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// no-op: placement is a logical decision with no PVE-side resource to clean up
}
