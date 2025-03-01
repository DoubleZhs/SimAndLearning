package utils

import (
	"sync"

	"gonum.org/v1/gonum/graph"
)

type accessCache struct {
	cache      map[accessCacheKey][]graph.Node
	cacheMutex sync.RWMutex
}

type accessCacheKey struct {
	from graph.Node
	lim  int
}

func newAccessCache() *accessCache {
	return &accessCache{
		cache: make(map[accessCacheKey][]graph.Node),
	}
}

func (c *accessCache) Get(from graph.Node, lim int) ([]graph.Node, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	key := accessCacheKey{from: from, lim: lim}
	if nodes, ok := c.cache[key]; ok {
		return nodes, true
	}
	return nil, false
}

func (c *accessCache) Set(from graph.Node, lim int, nodes []graph.Node) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	key := accessCacheKey{from: from, lim: lim}
	c.cache[key] = nodes
}
