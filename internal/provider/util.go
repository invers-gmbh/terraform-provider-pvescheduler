package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultMemWeight = 0.7
	defaultCpuWeight = 0.3
)

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

func buildAllowlist(nodes []string, topology map[string][]string, zone string) (map[string]bool, error) {
	if zone == "" {
		return toSet(nodes), nil
	}

	if len(topology) == 0 {
		return nil, fmt.Errorf("zone %q set but provider has no topology configured", zone)
	}

	members, ok := topology[zone]

	if !ok {
		known := slices.Sorted(maps.Keys(topology))
		return nil, fmt.Errorf("unknown zone %q, known zones: %s", zone, strings.Join(known, ", "))
	}
	global := toSet(nodes)
	allowlist := map[string]bool{}

	for _, n := range members {
		if len(global) == 0 || global[n] {
			allowlist[n] = true
		}
	}

	if len(allowlist) == 0 {
		return nil, fmt.Errorf("zone %q has no eligible nodes", zone)
	}

	return allowlist, nil
}

func zoneOf(topology map[string][]string, node string) types.String {
	var found []string
	for zone, members := range topology {
		if slices.Contains(members, node) {
			found = append(found, zone)
		}
	}
	if len(found) != 1 {
		return types.StringNull()
	}
	return types.StringValue(found[0])
}

func resolveWeights(memWeight, cpuWeight types.Float64) (float64, float64) {
	mem, cpu := defaultMemWeight, defaultCpuWeight
	if !memWeight.IsNull() && !memWeight.IsUnknown() {
		mem = memWeight.ValueFloat64()
	}
	if !cpuWeight.IsNull() && !cpuWeight.IsUnknown() {
		cpu = cpuWeight.ValueFloat64()
	}
	return mem, cpu
}
