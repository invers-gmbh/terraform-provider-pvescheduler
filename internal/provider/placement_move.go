package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithMoveState = &PlacementDynamicResource{}
var _ resource.ResourceWithMoveState = &PlacementResourceFixed{}

const privateKeyZonePending = "zone_pending"

type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func legacyPlacementSchema(ctx context.Context) *schema.Schema {
	var resp resource.SchemaResponse
	NewPlacementResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	return &resp.Schema
}

func readLegacyPlacement(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) (PlacementResourceModel, bool) {
	var src PlacementResourceModel
	if req.SourceTypeName != "pvescheduler_placement" ||
		!strings.HasSuffix(req.SourceProviderAddress, "/pvescheduler") ||
		req.SourceSchemaVersion != 0 {
		return src, false
	}
	if req.SourceState == nil {
		resp.Diagnostics.AddError("unable to read source state", "the state of pvescheduler_placement does not match its schema")
		return src, false
	}
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &src)...)
	resp.Diagnostics.Append(resp.TargetPrivate.SetKey(ctx, privateKeyZonePending, []byte("true"))...)
	return src, !resp.Diagnostics.HasError()
}

func resolvePendingZone(ctx context.Context, client *PveClient, nodeName types.String, zone *types.String, priv privateState, diags *diag.Diagnostics) {
	pending, d := priv.GetKey(ctx, privateKeyZonePending)
	diags.Append(d...)
	if pending == nil || client == nil {
		return
	}
	*zone = zoneOf(client.Topology, nodeName.ValueString())
	diags.Append(priv.SetKey(ctx, privateKeyZonePending, nil)...)
}

func (r *PlacementDynamicResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		SourceSchema: legacyPlacementSchema(ctx),
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			src, ok := readLegacyPlacement(ctx, req, resp)
			if !ok {
				return
			}
			target := PlacementDynamicResourceModel{
				ID:          src.ID,
				NodeName:    src.NodeName,
				Exclude:     src.Exclude,
				MemWeight:   src.MemWeight,
				CpuWeight:   src.CpuWeight,
				MemUsagePct: src.MemUsagePct,
				CpuUsagePct: src.CpuUsagePct,
				Zone:        types.StringNull(),
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
		},
	}}
}

func (r *PlacementResourceFixed) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		SourceSchema: legacyPlacementSchema(ctx),
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			src, ok := readLegacyPlacement(ctx, req, resp)
			if !ok {
				return
			}
			target := PlacementResourceFixedModel{
				ID:          src.ID,
				NodeName:    src.NodeName,
				MemUsagePct: src.MemUsagePct,
				CpuUsagePct: src.CpuUsagePct,
				Zone:        types.StringNull(),
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
		},
	}}
}
