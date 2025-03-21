package simulator

import (
	"fmt"
	"graphCA/element"
	"graphCA/utils"
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

// CreateStarRingGraph 创建一个星形环形混合路网结构，其中包含5个节点：A,B,C,D围成一个环，E作为中心
// 每两个相邻节点之间都有双向连接
// 参数:
//   - ringCellsPerDirection: 环形连接每个方向的元胞数量
//   - starCellsPerDirection: 星形连接每个方向的元胞数量
//   - initInterval: 红绿灯初始周期时长
//
// 返回:
//   - *simple.DirectedGraph: 创建的有向图
//   - map[int64]graph.Node: 图中所有节点的映射
//   - map[int64]*element.TrafficLightCell: 交通灯元胞映射
func CreateStarRingGraph(ringCellsPerDirection int, starCellsPerDirection int, initInterval int) (*simple.DirectedGraph, map[int64]graph.Node, map[int64]*element.TrafficLightCell) {
	// 参数验证
	if ringCellsPerDirection <= 0 {
		panic("ringCellsPerDirection must be positive")
	}
	if starCellsPerDirection <= 0 {
		panic("starCellsPerDirection must be positive")
	}
	if initInterval <= 0 {
		panic("initInterval must be positive")
	}

	// 创建新的有向图
	g := simple.NewDirectedGraph()

	// 创建存储节点的映射
	// 5个关键节点(A,B,C,D,E) + 环形连接元胞(8个方向 * ringCellsPerDirection) + 星形连接元胞(8个方向 * starCellsPerDirection)
	totalNodes := 5 + 8*ringCellsPerDirection + 8*starCellsPerDirection
	nodes := make(map[int64]graph.Node, totalNodes)
	lights := make(map[int64]*element.TrafficLightCell)

	// 节点ID计数器
	nextID := int64(0)

	// 创建5个关键节点: A,B,C,D围成一个环，E作为中心
	// 创建A、B、C、D、E基础节点
	nodeA := element.NewCommonCell(nextID, 5, 1.0)
	g.AddNode(nodeA)
	nodes[nextID] = nodeA
	nextID++

	nodeB := element.NewCommonCell(nextID, 5, 1.0)
	g.AddNode(nodeB)
	nodes[nextID] = nodeB
	nextID++

	nodeC := element.NewCommonCell(nextID, 5, 1.0)
	g.AddNode(nodeC)
	nodes[nextID] = nodeC
	nextID++

	nodeD := element.NewCommonCell(nextID, 5, 1.0)
	g.AddNode(nodeD)
	nodes[nextID] = nodeD
	nextID++

	// 创建E节点（中心节点）
	nodeE := element.NewCommonCell(nextID, 5, 1.0)
	g.AddNode(nodeE)
	nodes[nextID] = nodeE
	nextID++

	// 定义所有连接关系并指定每种连接类型的元胞数
	connections := []struct {
		from, to   graph.Node
		cellCount  int
		isToCenter bool // 标记是否是通向中心的路径
		lightIndex int  // 红绿灯相位索引，-1表示不是红绿灯节点
	}{
		// 环形连接（顺时针）
		{nodeA, nodeB, ringCellsPerDirection, false, -1}, // A -> B
		{nodeB, nodeC, ringCellsPerDirection, false, -1}, // B -> C
		{nodeC, nodeD, ringCellsPerDirection, false, -1}, // C -> D
		{nodeD, nodeA, ringCellsPerDirection, false, -1}, // D -> A

		// 环形连接（逆时针）
		{nodeB, nodeA, ringCellsPerDirection, false, -1}, // B -> A
		{nodeC, nodeB, ringCellsPerDirection, false, -1}, // C -> B
		{nodeD, nodeC, ringCellsPerDirection, false, -1}, // D -> C
		{nodeA, nodeD, ringCellsPerDirection, false, -1}, // A -> D

		// 星形连接（从外围到中心）
		{nodeA, nodeE, starCellsPerDirection, true, 0}, // A -> E
		{nodeB, nodeE, starCellsPerDirection, true, 1}, // B -> E
		{nodeC, nodeE, starCellsPerDirection, true, 2}, // C -> E
		{nodeD, nodeE, starCellsPerDirection, true, 3}, // D -> E

		// 星形连接（从中心到外围）
		{nodeE, nodeA, starCellsPerDirection, false, -1}, // E -> A
		{nodeE, nodeB, starCellsPerDirection, false, -1}, // E -> B
		{nodeE, nodeC, starCellsPerDirection, false, -1}, // E -> C
		{nodeE, nodeD, starCellsPerDirection, false, -1}, // E -> D
	}

	// 为每个连接创建元胞链
	for connectionIndex, conn := range connections {
		// 每个方向的中间元胞数量
		intermediateCount := conn.cellCount
		if intermediateCount < 0 {
			intermediateCount = 0
		}

		intermediateNodes := make([]graph.Node, intermediateCount)

		// 创建中间元胞
		for j := 0; j < intermediateCount; j++ {
			id := nextID

			// 检查是否创建红绿灯节点（通向中心的路径，且是最后一个元胞）
			var cell graph.Node
			if conn.isToCenter && j == intermediateCount-1 && conn.lightIndex >= 0 {
				// 计算相位区间 - 为每个方向提供四分之一周期的绿灯时间
				phaseStart := conn.lightIndex * (initInterval / 4)
				phaseEnd := (conn.lightIndex + 1) * (initInterval / 4)
				if conn.lightIndex == 3 {
					phaseEnd = initInterval // 确保最后一个相位正好覆盖到周期结束
				}
				phaseInterval := [2]int{phaseStart, phaseEnd}

				// 创建红绿灯节点
				light := element.NewTrafficLightCell(
					id,            // 节点ID
					5,             // 最大速度
					1.0,           // 容量
					initInterval,  // 周期长度
					phaseInterval, // 相位区间
				)

				// 随机设置初始计数，使红绿灯初始相位随机分布
				randomCount := 1
				if initInterval > 1 {
					randomCount = rand.IntN(initInterval-1) + 1
				}
				light.SetCount(randomCount)

				cell = light
				lights[id] = light
			} else {
				// 创建普通节点
				cell = element.NewCommonCell(id, 5, 1.0)
			}

			g.AddNode(cell)
			nodes[id] = cell
			intermediateNodes[j] = cell
			nextID++
		}

		// 连接起点和终点，中间通过所有中间元胞
		if len(intermediateNodes) > 0 {
			// 连接起点到第一个中间元胞
			g.SetEdge(simple.Edge{F: conn.from, T: intermediateNodes[0]})

			// 连接所有中间元胞
			for j := 0; j < len(intermediateNodes)-1; j++ {
				g.SetEdge(simple.Edge{F: intermediateNodes[j], T: intermediateNodes[j+1]})
			}

			// 连接最后一个中间元胞到终点
			g.SetEdge(simple.Edge{F: intermediateNodes[len(intermediateNodes)-1], T: conn.to})
		} else {
			// 如果没有中间元胞，直接连接起点到终点
			g.SetEdge(simple.Edge{F: conn.from, T: conn.to})
		}

		// 避免未使用变量警告
		_ = connectionIndex
	}

	return g, nodes, lights
}

// VerifyStarRingGraphConnectivity 验证星形环形图的强连通性
// 参数:
//   - g: 要验证的图
//
// 返回:
//   - bool: 图是否强连通
//   - []string: 连通性问题的详细信息（如果有）
func VerifyStarRingGraphConnectivity(g *simple.DirectedGraph) (bool, []string) {
	// 使用utils包中的IsStronglyConnected函数
	isStronglyConnected := utils.IsStronglyConnected(g)

	// 如果图已经是强连通的，直接返回
	if isStronglyConnected {
		return true, nil
	}

	// 如果图不是强连通的，进行更详细的分析
	problems := make([]string, 0)

	// 获取图中所有节点
	nodes := graph.NodesOf(g.Nodes())
	nodeCount := len(nodes)

	// 检查每对节点之间的可达性
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < nodeCount; j++ {
			if i != j {
				// 检查从节点i到节点j是否可达
				// 使用简单的BFS算法
				visited := make(map[int64]bool)
				queue := []graph.Node{nodes[i]}
				reachable := false

				for len(queue) > 0 && !reachable {
					node := queue[0]
					queue = queue[1:]

					if node.ID() == nodes[j].ID() {
						reachable = true
						break
					}

					// 将所有未访问的邻居节点加入队列
					neighbors := g.From(node.ID())
					for neighbors.Next() {
						neighbor := neighbors.Node()
						if !visited[neighbor.ID()] {
							visited[neighbor.ID()] = true
							queue = append(queue, neighbor)
						}
					}
				}

				// 如果不可达，记录问题
				if !reachable {
					problems = append(problems, fmt.Sprintf("节点 %d 到节点 %d 不可达", nodes[i].ID(), nodes[j].ID()))
				}
			}
		}
	}

	return false, problems
}
