package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NodeSchedulerDataSource struct{ client *PveClient }

type NodeSchedulerModel struct {
	NodeName    types.String  `tfsdk:"node_name"`
	MemUsagePct types.Number  `tfsdk:"memory_usage_pct"`
	CpuUsagePct types.Number  `tfsdk:"cpu_usage_pct"`
	Exclude     types.List    `tfsdk:"exclude"`
	MemWeight   types.Float64 `tfsdk:"memory_weight"`
	CpuWeight   types.Float64 `tfsdk:"cpu_weight"`
}

func NewNodeSchedulerDataSource() datasource.DataSource {
	return &NodeSchedulerDataSource{}
}

func (d *NodeSchedulerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

func (d *NodeSchedulerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Evaluates all PVE nodes on every plan and returns the currently least-loaded one. Use pvescheduler_placement to lock in a decision instead.",
		Attributes: map[string]schema.Attribute{
			"node_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the selected Proxmox node.",
			},
			"memory_usage_pct": schema.NumberAttribute{
				Computed:    true,
				Description: "Memory utilization of the selected node as a percentage (0-100).",
			},
			"cpu_usage_pct": schema.NumberAttribute{
				Computed:    true,
				Description: "CPU utilization of the selected node as a percentage (0-100).",
			},
			"exclude": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Node names to exclude from scheduling.",
			},
			"memory_weight": schema.Float64Attribute{
				Optional:    true,
				Description: "Weight applied to memory utilization when scoring nodes (default 0.7).",
			},
			"cpu_weight": schema.Float64Attribute{
				Optional:    true,
				Description: "Weight applied to CPU utilization when scoring nodes (default 0.3).",
			},
		},
	}
}

func (d *NodeSchedulerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*PveClient)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *NodeSchedulerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state NodeSchedulerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pveNodes, err := d.client.GetNodes()
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch PVE nodes", err.Error())
		return
	}

	allowlist := toSet(d.client.Nodes)
	exclude := buildExcludeSet(ctx, state.Exclude, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	memWeight, cpuWeight := resolveWeights(state.MemWeight, state.CpuWeight)

	best, err := selectNode(pveNodes, allowlist, exclude, memWeight, cpuWeight)
	if err != nil {
		resp.Diagnostics.AddError("node selection failed", err.Error())
		return
	}

	state.NodeName = types.StringValue(best.Node)
	state.MemUsagePct = types.NumberValue(bigFloat((best.Mem / best.MaxMem) * 100))
	state.CpuUsagePct = types.NumberValue(bigFloat(best.Cpu * 100))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
