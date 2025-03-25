package utils

import (
	"simAndLearning/element"
	"sync"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/traverse"
)

var (
	accessCacheInstance = newAccessCache()
)

func Accessible(g *simple.DirectedGraph, from, to graph.Node) bool {
	dfs := traverse.DepthFirst{}
	found := false

	dfs.Walk(g, from, func(n graph.Node) bool {
		if n.ID() == to.ID() {
			found = true
			return false // Stop the traversal
		}
		return true // Continue the traversal
	})

	return found
}

func AccessibleNodesWithinLim(g graph.Graph, from graph.Node, lim int) []graph.Node {
	// Check cache first
	if cachedNodes, ok := accessCacheInstance.Get(from, lim); ok {
		return cachedNodes
	}

	visited := make(map[int64]bool)
	var visitedMu sync.Mutex
	result := []graph.Node{}
	var resultMu sync.Mutex
	var wg sync.WaitGroup

	var dfs func(node graph.Node, distance int)

	dfs = func(node graph.Node, distance int) {
		defer wg.Done()
		if distance > lim {
			return
		}

		visitedMu.Lock()
		if visited[node.ID()] {
			visitedMu.Unlock()
			return
		}
		visited[node.ID()] = true
		visitedMu.Unlock()

		if _, ok := node.(element.Cell); ok {
			resultMu.Lock()
			result = append(result, node)
			resultMu.Unlock()
		}

		for _, neighbor := range graph.NodesOf(g.From(node.ID())) {
			switch n := neighbor.(type) {
			case element.Cell:
				wg.Add(1)
				go dfs(n, distance+1)
			case *element.Link:
				wg.Add(1)
				go dfs(n, distance+n.Length())
			}
		}
	}

	wg.Add(1)
	go dfs(from, 0)

	wg.Wait()

	// Store result in cache
	accessCacheInstance.Set(from, lim, result)

	return result
}

func AccessibleNodesWithinRange(g *simple.DirectedGraph, from graph.Node, lim1, lim2 int) []graph.Node {
	var nodesInLim1, nodesInLim2 []graph.Node
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		nodesInLim1 = AccessibleNodesWithinLim(g, from, lim1)
	}()

	go func() {
		defer wg.Done()
		nodesInLim2 = AccessibleNodesWithinLim(g, from, lim2)
	}()
	wg.Wait()

	nodesInLim1Map := make(map[int64]struct{})
	for _, node := range nodesInLim1 {
		nodesInLim1Map[node.ID()] = struct{}{}
	}

	var result []graph.Node
	for _, node := range nodesInLim2 {
		if _, found := nodesInLim1Map[node.ID()]; !found {
			result = append(result, node)
		}
	}

	return result
}
