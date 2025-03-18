package simulator

import (
	"graphCA/element"
	"math"

	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// CreateCycleGraph 创建一个环形路网图
//
// 参数:
//   - cellNum: 单元格总数
//   - trafficLightInterval: 红绿灯间隔（每隔多少个单元格放置一个红绿灯）
//   - initInterval: 红绿灯初始周期时长
//
// 返回:
//   - *simple.DirectedGraph: 创建的有向图
//   - map[int64]graph.Node: 图中所有节点的映射
//   - map[int64]*element.TrafficLightCell: 红绿灯节点的映射
func CreateCycleGraph(cellNum int, trafficLightInterval int, initInterval int) (*simple.DirectedGraph, map[int64]graph.Node, map[int64]*element.TrafficLightCell) {
	// 参数验证
	if cellNum <= 0 {
		panic("cellNum must be positive")
	}
	if trafficLightInterval <= 0 {
		panic("trafficLightInterval must be positive")
	}
	if initInterval <= 0 {
		panic("initInterval must be positive")
	}

	// 创建新的有向图
	g := simple.NewDirectedGraph()

	// 创建存储节点的映射
	nodes := make(map[int64]graph.Node, cellNum)
	lights := make(map[int64]*element.TrafficLightCell)

	// 创建所有节点（单元格）
	trafficLightRatioCount := 0
	for i := 0; i < cellNum; i++ {
		var node graph.Node

		// 根据间隔创建红绿灯单元格或普通单元格
		if i%trafficLightInterval == 0 {
			// 交替设置绿灯比例，一个0.3，一个0.7
			greenRatio := 0.0
			if trafficLightRatioCount%2 == 0 {
				greenRatio = 0.3
			} else {
				greenRatio = 0.7
			}

			// 计算相位区间
			greenPhaseEnd := int(math.Round(float64(initInterval) * greenRatio))
			phaseInterval := [2]int{0, greenPhaseEnd}

			// 创建红绿灯单元格
			light := element.NewTrafficLightCell(
				int64(i),      // 单元格ID
				5,             // 最大速度
				1.0,           // 容量
				initInterval,  // 周期长度
				phaseInterval, // 相位区间
			)

			// 随机设置初始计数，使红绿灯初始相位随机分布
			// 确保计数值在1到interval之间，避免无效值
			randomCount := 1
			if initInterval > 1 {
				randomCount = rand.IntN(initInterval-1) + 1 // 生成1到initInterval之间的随机数
			}
			light.SetCount(randomCount)

			node = light
			lights[int64(i)] = light
			trafficLightRatioCount++
		} else {
			// 创建普通单元格
			node = element.NewCommonCell(
				int64(i), // 单元格ID
				5,        // 最大速度
				1.0,      // 容量
			)
		}

		// 将节点添加到图和映射中
		g.AddNode(node)
		nodes[int64(i)] = node
	}

	// 创建边（连接相邻节点）
	for i := 0; i < cellNum-1; i++ {
		g.SetEdge(simple.Edge{F: nodes[int64(i)], T: nodes[int64(i+1)]})
	}
	// 连接最后一个节点到第一个节点，形成环形
	g.SetEdge(simple.Edge{F: nodes[int64(cellNum-1)], T: nodes[int64(0)]})

	return g, nodes, lights
}
