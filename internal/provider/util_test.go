package provider

import (
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
