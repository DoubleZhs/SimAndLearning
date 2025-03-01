package utils

import (
	"sync"

	"gonum.org/v1/gonum/graph"
)

var kpCache = newKPathsCache()

type kPathsCache struct {
	cache       map[[2]graph.Node][][]graph.Node
	pathLengths map[[2]graph.Node][]float64
	cacheMutex  sync.RWMutex
}

func newKPathsCache() *kPathsCache {
	return &kPathsCache{
		cache:       make(map[[2]graph.Node][][]graph.Node),
		pathLengths: make(map[[2]graph.Node][]float64),
	}
}

func (c *kPathsCache) GetK(from, to graph.Node) ([][]graph.Node, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	if nodes, ok := c.cache[[2]graph.Node{from, to}]; ok {
		return nodes, true
	}
	return nil, false
}

func (c *kPathsCache) SetK(from, to graph.Node, paths [][]graph.Node, lengths []float64) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache[[2]graph.Node{from, to}] = paths
	c.pathLengths[[2]graph.Node{from, to}] = lengths
}
