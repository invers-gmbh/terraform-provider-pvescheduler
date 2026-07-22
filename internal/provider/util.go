package provider

import (
	"context"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func bigFloat(f float64) *big.Float {
	return big.NewFloat(f)
}

func buildExcludeSet(ctx context.Context, exclude types.List, diags *diag.Diagnostics) map[string]bool {
	excludeSet := map[string]bool{}
	if !exclude.IsNull() && !exclude.IsUnknown() {
		var list []string
		diags.Append(exclude.ElementsAs(ctx, &list, false)...)
		for _, n := range list {
			excludeSet[n] = true
		}
	}
	return excludeSet
}

func toSet(items []string) map[string]bool {
	s := map[string]bool{}
	for _, v := range items {
		s[v] = true
	}
	return s
}

func resolveWeights(memWeight, cpuWeight types.Float64) (float64, float64) {
	mem, cpu := 0.7, 0.3
	if !memWeight.IsNull() && !memWeight.IsUnknown() {
		mem = memWeight.ValueFloat64()
	}
	if !cpuWeight.IsNull() && !cpuWeight.IsUnknown() {
		cpu = cpuWeight.ValueFloat64()
	}
	return mem, cpu
}
