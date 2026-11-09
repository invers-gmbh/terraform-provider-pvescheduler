package provider

import (
	"maps"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestToSet_Empty(t *testing.T) {
	s := toSet(nil)
	if len(s) != 0 {
		t.Errorf("expected empty set, got %v", s)
	}
}

func TestToSet_Items(t *testing.T) {
	s := toSet([]string{"a", "b", "c"})
	for _, k := range []string{"a", "b", "c"} {
		if !s[k] {
			t.Errorf("expected %q in set", k)
		}
	}
	if len(s) != 3 {
		t.Errorf("expected 3 items, got %d", len(s))
	}
}

func TestBuildAllowlist_EmptyZoneString(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	zones := map[string][]string{
		"foo": {"a"},
		"bar": {"b"},
		"baz": {"c"},
	}
	zone := ""
	allowlist, err := buildAllowlist(nodes, zones, zone)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := slices.Sorted(maps.Keys(allowlist))
	if !slices.Equal(got, nodes) {
		t.Errorf("expected %v, got %v", nodes, got)
	}
}

func TestBuildAllowlist_EmptyTopology(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	zones := map[string][]string{}
	zone := "foo"

	_, err := buildAllowlist(nodes, zones, zone)

	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAllowlist_ExplicitValues(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	zones := map[string][]string{
		"foo": {"a"},
		"bar": {"b"},
		"baz": {"c"},
	}
	zone := "foo"

	allowlist, err := buildAllowlist(nodes, zones, zone)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !slices.Contains(slices.Sorted(maps.Keys(allowlist)), "a") {
		t.Errorf("Expecet \"a\" to be in allowlist, got: %v", allowlist)
	}

	if len(allowlist) != 1 {
		t.Errorf("Expected allowlist of one, got: %d", len(allowlist))
	}
}

func TestBuildAllowlist_NoEligibleNodes(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	zones := map[string][]string{
		"foo": {"d"},
		"bar": {"b"},
		"baz": {"c"},
	}
	zone := "foo"

	allowlist, err := buildAllowlist(nodes, zones, zone)

	if err == nil {
		t.Fatalf("expected error, got allowlist: %v", allowlist)
	}
}

func TestBuildAllowlist_UnknownZone(t *testing.T) {
	nodes := []string{"a", "b", "c"}
	zones := map[string][]string{
		"foo": {"a"},
	}
	zone := "bar"

	allowlist, err := buildAllowlist(nodes, zones, zone)

	if err == nil {
		t.Fatalf("expected error, got allowlist: %v", allowlist)
	}
}

func TestBuildAllowlist_NoGlobalAllowlist(t *testing.T) {
	zones := map[string][]string{
		"foo": {"a", "b"},
		"bar": {"c"},
	}
	zone := "foo"

	allowlist, err := buildAllowlist(nil, zones, zone)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a", "b"}
	got := slices.Sorted(maps.Keys(allowlist))
	if !slices.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestBuildAllowlist_PartialIntersection(t *testing.T) {
	nodes := []string{"a", "b"}
	zones := map[string][]string{
		"foo": {"a", "c"},
	}
	zone := "foo"

	allowlist, err := buildAllowlist(nodes, zones, zone)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a"}
	got := slices.Sorted(maps.Keys(allowlist))
	if !slices.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestResolveWeights_Defaults(t *testing.T) {
	mem, cpu := resolveWeights(types.Float64Null(), types.Float64Null())
	if mem != 0.7 {
		t.Errorf("expected mem 0.7, got %f", mem)
	}
	if cpu != 0.3 {
		t.Errorf("expected cpu 0.3, got %f", cpu)
	}
}

func TestResolveWeights_ExplicitValues(t *testing.T) {
	mem, cpu := resolveWeights(types.Float64Value(0.5), types.Float64Value(0.5))
	if mem != 0.5 {
		t.Errorf("expected mem 0.5, got %f", mem)
	}
	if cpu != 0.5 {
		t.Errorf("expected cpu 0.5, got %f", cpu)
	}
}

func TestResolveWeights_UnknownFallsBack(t *testing.T) {
	mem, cpu := resolveWeights(types.Float64Unknown(), types.Float64Unknown())
	if mem != 0.7 {
		t.Errorf("expected mem default 0.7, got %f", mem)
	}
	if cpu != 0.3 {
		t.Errorf("expected cpu default 0.3, got %f", cpu)
	}
}
