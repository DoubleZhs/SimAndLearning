package utils

import (
	"sync"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func IsStronglyConnected(g *simple.DirectedGraph) bool {
	it := g.Nodes()
	if !it.Next() {
		return false
	}
	startNode := it.Node()

	if !isFullyReachable(g, startNode) {
		return false
	}

	reversedGraph := reverseGraph(g)

	return isFullyReachable(reversedGraph, startNode)
}

func isFullyReachable(g *simple.DirectedGraph, startNode graph.Node) bool {
	var visited sync.Map
	var queue []graph.Node
	visitedCount := 0

	queue = append(queue, startNode)
	visited.Store(startNode.ID(), true)
	visitedCount++

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		neighbors := g.From(node.ID())
		for neighbors.Next() {
			neighbor := neighbors.Node()
			if _, ok := visited.Load(neighbor.ID()); !ok {
				visited.Store(neighbor.ID(), true)
				visitedCount++
				queue = append(queue, neighbor)
			}
		}
	}

	return visitedCount == g.Nodes().Len()
}

func reverseGraph(g *simple.DirectedGraph) *simple.DirectedGraph {
	reversed := simple.NewDirectedGraph()

	nodes := g.Nodes()
	for nodes.Next() {
		reversed.AddNode(nodes.Node())
	}

	edges := g.Edges()
	for edges.Next() {
		edge := edges.Edge()
		reversed.SetEdge(simple.Edge{
			F: edge.To(),
			T: edge.From(),
		})
	}

	return reversed
}
