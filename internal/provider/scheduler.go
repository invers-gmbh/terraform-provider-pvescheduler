package provider

import "fmt"

func selectNode(nodes []PveNode, allowlist map[string]bool, exclude map[string]bool, memWeight, cpuWeight float64) (*PveNode, error) {
	score := func(n *PveNode) float64 {
		return memWeight*(n.Mem/n.MaxMem) + cpuWeight*n.Cpu
	}

	var best *PveNode
	for i := range nodes {
		n := &nodes[i]
		if n.Status != "online" || n.MaxMem == 0 {
			continue
		}
		if exclude[n.Node] {
			continue
		}
		if len(allowlist) > 0 && !allowlist[n.Node] {
			continue
		}
		if best == nil || score(n) < score(best) {
			best = n
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no eligible nodes found (check that nodes are online and not excluded)")
	}
	return best, nil
}
