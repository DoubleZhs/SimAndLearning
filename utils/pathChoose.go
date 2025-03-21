package utils

import (
	"graphCA/config"
	"math"
	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// PathFinder 定义了查找路径的函数类型
type PathFinder func(g *simple.DirectedGraph, origin, destination graph.Node) ([]graph.Node, float64, error)

// GetPathFinder 根据配置返回相应的路径查找函数
func GetPathFinder() PathFinder {
	cfg := config.GetConfig()

	switch cfg.Path.PathMethod {
	case "shortest":
		return ShortestPath
	case "random":
		return RandomPath
	case "kShortest":
		return func(g *simple.DirectedGraph, origin, destination graph.Node) ([]graph.Node, float64, error) {
			return ChooseFromKShortestPaths(g, origin, destination, cfg.Path.KShortest.K,
				cfg.Path.KShortest.SelectionStrategy, cfg.Path.KShortest.LengthWeightFactor)
		}
	default:
		// 默认使用最短路径
		return ShortestPath
	}
}

// ChooseFromKShortestPaths 从k条最短路径中选择一条
func ChooseFromKShortestPaths(g *simple.DirectedGraph, origin, destination graph.Node, k int,
	strategy string, weightFactor float64) ([]graph.Node, float64, error) {

	// 获取k条最短路径
	paths, err := KShortestPaths(g, origin, destination, k)
	if err != nil {
		return nil, -1, err
	}

	// 如果只有一条路径，直接返回
	if len(paths) == 1 {
		return paths[0], calPathLength(paths[0]), nil
	}

	// 根据策略选择路径
	switch strategy {
	case "random":
		// 随机选择一条路径
		selectedIndex := rand.IntN(len(paths))
		return paths[selectedIndex], calPathLength(paths[selectedIndex]), nil

	case "weighted":
		// 基于路径长度进行加权选择，路径越短权重越大
		weights := make([]float64, len(paths))
		totalWeight := 0.0

		// 计算每条路径的长度和权重
		lengths := make([]float64, len(paths))
		maxLength := 0.0

		for i, path := range paths {
			lengths[i] = calPathLength(path)
			if lengths[i] > maxLength {
				maxLength = lengths[i]
			}
		}

		// 计算权重：使用指数函数使短路径获得更高权重
		for i, length := range lengths {
			// 归一化长度（值越小表示路径越短）
			normalizedLength := length / maxLength
			// 计算权重：越短的路径权重越大
			weights[i] = math.Exp(-weightFactor * normalizedLength)
			totalWeight += weights[i]
		}

		// 随机选择（加权）
		r := rand.Float64() * totalWeight
		cumulativeWeight := 0.0
		selectedIndex := 0

		for i, weight := range weights {
			cumulativeWeight += weight
			if r <= cumulativeWeight {
				selectedIndex = i
				break
			}
		}

		return paths[selectedIndex], lengths[selectedIndex], nil

	default:
		// 默认返回最短的那条路径
		return paths[0], calPathLength(paths[0]), nil
	}
}
