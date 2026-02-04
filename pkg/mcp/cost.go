package mcp

import (
	"strings"
)

// CostProfile maps tool names to credit costs with optional wildcard support.
type CostProfile struct {
	// ToolCosts maps exact tool names (or glob patterns like "db_*") to costs.
	ToolCosts map[string]int

	// DefaultCost is returned when no matching entry is found.
	DefaultCost int
}

// Lookup returns the credit cost for the given tool name.
//
// Matching order:
//  1. Exact match in ToolCosts.
//  2. Wildcard match — patterns ending in "*" are matched as prefixes.
//     The longest matching pattern wins.
//  3. DefaultCost.
func (cp *CostProfile) Lookup(toolName string) int {
	if cp == nil {
		return 0
	}

	// 1. Exact match.
	if cost, ok := cp.ToolCosts[toolName]; ok {
		return cost
	}

	// 2. Wildcard match (longest prefix wins).
	bestLen := -1
	bestCost := cp.DefaultCost
	for pattern, cost := range cp.ToolCosts {
		if !strings.HasSuffix(pattern, "*") {
			continue
		}
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(toolName, prefix) && len(prefix) > bestLen {
			bestLen = len(prefix)
			bestCost = cost
		}
	}
	if bestLen >= 0 {
		return bestCost
	}

	// 3. Default.
	return cp.DefaultCost
}
