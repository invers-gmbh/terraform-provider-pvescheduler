package provider

import (
	"testing"
)

func makeNode(name, status string, mem, maxMem, cpu float64) PveNode {
	return PveNode{Node: name, Status: status, Mem: mem, MaxMem: maxMem, Cpu: cpu}
}

func TestSelectNode_PicksLowestScore(t *testing.T) {
	nodes := []PveNode{
		makeNode("heavy", "online", 8e9, 10e9, 0.9), // score = 0.7*0.8 + 0.3*0.9 = 0.83
		makeNode("light", "online", 2e9, 10e9, 0.1), // score = 0.7*0.2 + 0.3*0.1 = 0.17
	}
	best, err := selectNode(nodes, nil, nil, 0.7, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Node != "light" {
		t.Errorf("expected light, got %s", best.Node)
	}
}

func TestSelectNode_SkipsOfflineNodes(t *testing.T) {
	nodes := []PveNode{
		makeNode("offline", "offline", 1e9, 10e9, 0.05),
		makeNode("online", "online", 5e9, 10e9, 0.5),
	}
	best, err := selectNode(nodes, nil, nil, 0.7, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Node != "online" {
		t.Errorf("expected online, got %s", best.Node)
	}
}

func TestSelectNode_SkipsZeroMaxMem(t *testing.T) {
	nodes := []PveNode{
		makeNode("broken", "online", 0, 0, 0.0),
		makeNode("good", "online", 5e9, 10e9, 0.5),
	}
	best, err := selectNode(nodes, nil, nil, 0.7, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Node != "good" {
		t.Errorf("expected good, got %s", best.Node)
	}
}

func TestSelectNode_RespectsExclude(t *testing.T) {
	nodes := []PveNode{
		makeNode("excluded", "online", 1e9, 10e9, 0.05),
		makeNode("allowed", "online", 5e9, 10e9, 0.5),
	}
	exclude := map[string]bool{"excluded": true}
	best, err := selectNode(nodes, nil, exclude, 0.7, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Node != "allowed" {
		t.Errorf("expected allowed, got %s", best.Node)
	}
}

func TestSelectNode_RespectsAllowlist(t *testing.T) {
	nodes := []PveNode{
		makeNode("not-allowed", "online", 1e9, 10e9, 0.05),
		makeNode("allowed", "online", 5e9, 10e9, 0.5),
	}
	allowlist := map[string]bool{"allowed": true}
	best, err := selectNode(nodes, allowlist, nil, 0.7, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Node != "allowed" {
		t.Errorf("expected allowed, got %s", best.Node)
	}
}

func TestSelectNode_EmptyAllowlistAllowsAll(t *testing.T) {
	nodes := []PveNode{
		makeNode("a", "online", 1e9, 10e9, 0.1),
		makeNode("b", "online", 9e9, 10e9, 0.9),
	}
	best, err := selectNode(nodes, map[string]bool{}, nil, 0.7, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Node != "a" {
		t.Errorf("expected a, got %s", best.Node)
	}
}

func TestSelectNode_NoEligibleNodes(t *testing.T) {
	nodes := []PveNode{
		makeNode("offline", "offline", 1e9, 10e9, 0.1),
	}
	_, err := selectNode(nodes, nil, nil, 0.7, 0.3)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSelectNode_AllExcluded(t *testing.T) {
	nodes := []PveNode{
		makeNode("a", "online", 1e9, 10e9, 0.1),
		makeNode("b", "online", 2e9, 10e9, 0.2),
	}
	exclude := map[string]bool{"a": true, "b": true}
	_, err := selectNode(nodes, nil, exclude, 0.7, 0.3)
	if err == nil {
		t.Fatal("expected error when all nodes excluded")
	}
}
