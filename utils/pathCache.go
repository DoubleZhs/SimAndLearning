package utils

import (
	"sync"

	"gonum.org/v1/gonum/graph"
)

var pCache = newPathCache()

type pathCache struct {
	cache      map[[2]graph.Node][]graph.Node
	pathLength map[[2]graph.Node]float64
	cacheMutex sync.RWMutex
}

func newPathCache() *pathCache {
	return &pathCache{
		cache:      make(map[[2]graph.Node][]graph.Node),
		pathLength: make(map[[2]graph.Node]float64),
	}
}

func (c *pathCache) Get(from, to graph.Node) ([]graph.Node, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	if nodes, ok := c.cache[[2]graph.Node{from, to}]; ok {
		return nodes, true
	}
	return nil, false
}

func (c *pathCache) Set(from, to graph.Node, nodes []graph.Node, length float64) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache[[2]graph.Node{from, to}] = nodes
	c.pathLength[[2]graph.Node{from, to}] = length
}
