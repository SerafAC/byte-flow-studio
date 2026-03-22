package pipeline

import (
	"fmt"

	"byteflow-studio/internal/workflow"
)

// ValidateGraph validates the DAG formed by blocks and connections.
// It performs:
//  1. Port type compatibility check
//  2. Cycle detection via DFS
//  3. Topological sort (Kahn's algorithm)
//  4. At-least-one Input→Analysis path check
//
// portTypes maps blockID → portID → DataType for all declared ports.
// Returns the topological order of block IDs, or an error.
func ValidateGraph(
	blocks []workflow.BlockDef,
	conns []workflow.ConnectionDef,
	portTypes map[string]map[string]DataType,
) ([]string, error) {
	// Check port type compatibility
	for _, c := range conns {
		srcType, ok1 := portTypeOf(portTypes, c.FromBlockID, c.FromPortID)
		dstType, ok2 := portTypeOf(portTypes, c.ToBlockID, c.ToPortID)
		if ok1 && ok2 && srcType != dstType {
			return nil, fmt.Errorf("incompatible port types: %s → %s (connection %s)", srcType, dstType, c.ID)
		}
	}

	// Build adjacency list and in-degree map for Kahn's algorithm
	blockIDs := make([]string, 0, len(blocks))
	for _, b := range blocks {
		blockIDs = append(blockIDs, b.ID)
	}

	// Build block category map
	catMap := map[string]BlockCategory{}
	for _, b := range blocks {
		catMap[b.ID] = BlockCategory(b.Category)
	}

	adj := map[string][]string{}
	inDeg := map[string]int{}
	for _, id := range blockIDs {
		adj[id] = []string{}
		inDeg[id] = 0
	}
	for _, c := range conns {
		adj[c.FromBlockID] = append(adj[c.FromBlockID], c.ToBlockID)
		inDeg[c.ToBlockID]++
	}

	// Kahn's topological sort
	queue := []string{}
	for _, id := range blockIDs {
		if inDeg[id] == 0 {
			queue = append(queue, id)
		}
	}

	order := []string{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		for _, next := range adj[node] {
			inDeg[next]--
			if inDeg[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(order) != len(blockIDs) {
		return nil, fmt.Errorf("cycle detected in connection graph")
	}

	// Check that at least one Input→Analysis path exists
	if !hasInputToAnalysisPath(blockIDs, conns, catMap) {
		return nil, fmt.Errorf("no valid input-to-analysis path found")
	}

	return order, nil
}

// portTypeOf returns the DataType for a given block+port, if known.
func portTypeOf(portTypes map[string]map[string]DataType, blockID, portID string) (DataType, bool) {
	if ports, ok := portTypes[blockID]; ok {
		if dt, ok2 := ports[portID]; ok2 {
			return dt, true
		}
	}
	return "", false
}

// hasInputToAnalysisPath checks that at least one Input block can reach at least one Analysis block.
func hasInputToAnalysisPath(blockIDs []string, conns []workflow.ConnectionDef, catMap map[string]BlockCategory) bool {
	// Build forward adjacency
	adj := map[string][]string{}
	for _, id := range blockIDs {
		adj[id] = []string{}
	}
	for _, c := range conns {
		adj[c.FromBlockID] = append(adj[c.FromBlockID], c.ToBlockID)
	}

	// DFS from each Input block; check if any Analysis block is reachable
	for _, id := range blockIDs {
		if catMap[id] != CategoryInput {
			continue
		}
		if dfsReachesAnalysis(id, adj, catMap, map[string]bool{}) {
			return true
		}
	}
	return false
}

func dfsReachesAnalysis(cur string, adj map[string][]string, catMap map[string]BlockCategory, visited map[string]bool) bool {
	if visited[cur] {
		return false
	}
	visited[cur] = true
	if catMap[cur] == CategoryAnalysis {
		return true
	}
	for _, next := range adj[cur] {
		if dfsReachesAnalysis(next, adj, catMap, visited) {
			return true
		}
	}
	return false
}
