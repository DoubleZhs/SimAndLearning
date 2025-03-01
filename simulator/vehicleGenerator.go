package simulator

import (
	"graphCA/config"
	"graphCA/element"
	"graphCA/utils"
	"sync"
	"sync/atomic"

	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// getNextVehicleID 获取下一个可用的车辆ID
// 使用原子操作确保线程安全
func getNextVehicleID() int64 {
	return atomic.AddInt64(&numVehicleGenerated, 1)
}

// randomVelocity 生成随机初始速度 (1-3)
func randomVelocity() int {
	return 1 + rand.IntN(2)
}

// randomAcceleration 生成随机加速度 (1-4)
func randomAcceleration() int {
	return 1 + rand.IntN(3)
}

// randomSlowingProbability 生成随机减速概率 (0-0.5)
func randomSlowingProbability() float64 {
	return rand.Float64() / 2.0
}

// randomLowVelocity 随机生成较低基础速度
func randomLowVelocity() int {
	return 1 + rand.IntN(2)
}

// randomHighVelocity 随机生成较高基础速度
func randomHighVelocity() int {
	return 1 + rand.IntN(3)
}

// InitFixedVehicle 初始化固定数量的车辆
// 创建n个闭环车辆并将其添加到等待队列
// params:
//   - n: 要创建的车辆数量
//   - g: 路网图
//   - nodes: 可用节点列表
//   - traceNodes: 用于追踪的节点列表
func InitFixedVehicle(n int, g *simple.DirectedGraph, nodes []graph.Node, traceNodes []graph.Node) {
	if n <= 0 || len(nodes) == 0 {
		return // 避免无效输入
	}

	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	traceEnabled := cfg != nil && cfg.Trace.Enabled

	var wg sync.WaitGroup
	wg.Add(n)

	// 使用互斥锁保护traceNodes
	var traceNodesMutex sync.Mutex

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			// 从nodes中随机选择一个作为起点
			oCell := nodes[rand.IntN(len(nodes))]

			// 获取合适距离范围内的终点
			minLength, maxLength := TripDistanceRange()
			allowedDCells := utils.AccessibleNodesWithinRange(g, oCell, minLength, maxLength)

			// 如果没有合适的终点，返回
			if len(allowedDCells) == 0 {
				return
			}

			// 从可达节点中随机选择一个作为终点
			dCell := allowedDCells[rand.IntN(len(allowedDCells))]

			// 仅在启用轨迹记录时更新追踪节点列表
			if traceEnabled {
				traceNodesMutex.Lock()
				traceNodes = addNodeToTraceNodesIfMissing(traceNodes, oCell)
				traceNodes = addNodeToTraceNodesIfMissing(traceNodes, dCell)
				traceNodesMutex.Unlock()
			}

			// 创建新车辆
			vehicle := element.NewVehicle(
				getNextVehicleID(),
				randomVelocity(),
				randomAcceleration(),
				1.0, // 车辆长度
				randomSlowingProbability(),
				true, // ClosedVehicle = true，循环行驶
			)

			// 设置起点和终点
			ok, err := vehicle.SetOD(g, oCell, dCell)
			if !ok || err != nil {
				return // 设置失败，跳过此车辆
			}

			// 计算最短路径
			path, _, err := utils.ShortestPath(g, oCell, dCell)
			if err != nil {
				return // 路径计算失败，跳过此车辆
			}

			// 设置路径
			ok, err = vehicle.SetPath(path)
			if !ok || err != nil {
				return // 路径设置失败，跳过此车辆
			}

			// 将车辆加入缓冲区
			vehicle.BufferIn(0)

			// 仅在启用轨迹记录时设置检查点
			if traceEnabled {
				SetupVehicleCheckpoints(vehicle, 0) // 使用0表示使用默认间隔
			}

			// 添加到等待队列
			waitingVehiclesMutex.Lock()
			waitingVehicles[vehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()

			// 更新等待车辆计数
			atomic.AddInt64(&numVehiclesWaiting, 1)
		}()
	}
	wg.Wait()
}

// GenerateScheduleVehicle 按照给定时间生成计划车辆
// 创建n个非闭环车辆并将其添加到等待队列
// params:
//   - simTime: 当前模拟时间
//   - n: 要创建的车辆数量
//   - g: 路网图
//   - nodes: 可用节点列表
//   - traceNodes: 用于追踪的节点列表
func GenerateScheduleVehicle(simTime, n int, g *simple.DirectedGraph, nodes []graph.Node, traceNodes []graph.Node) {
	if n <= 0 || len(nodes) == 0 {
		return // 避免无效输入
	}

	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	traceEnabled := cfg != nil && cfg.Trace.Enabled

	var wg sync.WaitGroup
	wg.Add(n)

	// 使用互斥锁保护traceNodes
	var traceNodesMutex sync.Mutex

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			// 从nodes中随机选择一个作为起点
			oCell := nodes[rand.IntN(len(nodes))]

			// 获取合适距离范围内的终点
			minLength, maxLength := TripDistanceRange()
			allowedDCells := utils.AccessibleNodesWithinRange(g, oCell, minLength, maxLength)

			// 如果没有合适的终点，返回
			if len(allowedDCells) == 0 {
				return
			}

			// 从可达节点中随机选择一个作为终点
			dCell := allowedDCells[rand.IntN(len(allowedDCells))]

			// 仅在启用轨迹记录时更新追踪节点列表
			if traceEnabled {
				traceNodesMutex.Lock()
				traceNodes = addNodeToTraceNodesIfMissing(traceNodes, oCell)
				traceNodes = addNodeToTraceNodesIfMissing(traceNodes, dCell)
				traceNodesMutex.Unlock()
			}

			// 创建新车辆
			vehicle := element.NewVehicle(
				getNextVehicleID(),
				randomVelocity(),
				randomAcceleration(),
				1.0, // 车辆长度
				randomSlowingProbability(),
				false, // ClosedVehicle = false，完成后离开系统
			)

			// 设置起点和终点
			ok, err := vehicle.SetOD(g, oCell, dCell)
			if !ok || err != nil {
				return // 设置失败，跳过此车辆
			}

			// 计算最短路径
			path, _, err := utils.ShortestPath(g, oCell, dCell)
			if err != nil {
				return // 路径计算失败，跳过此车辆
			}

			// 设置路径
			ok, err = vehicle.SetPath(path)
			if !ok || err != nil {
				return // 路径设置失败，跳过此车辆
			}

			// 将车辆加入缓冲区
			vehicle.BufferIn(simTime)

			// 仅在启用轨迹记录时设置检查点
			if traceEnabled {
				SetupVehicleCheckpoints(vehicle, 0) // 使用0表示使用默认间隔
			}

			// 添加到等待队列
			waitingVehiclesMutex.Lock()
			waitingVehicles[vehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()

			// 更新等待车辆计数
			atomic.AddInt64(&numVehiclesWaiting, 1)
		}()
	}
	wg.Wait()
}

// addNodeToTraceNodesIfMissing 如果节点不在追踪列表中，则添加它
// 注意：此函数不是线程安全的，调用方必须持有适当的锁
func addNodeToTraceNodesIfMissing(traceNodes []graph.Node, node graph.Node) []graph.Node {
	for _, existingNode := range traceNodes {
		if existingNode.ID() == node.ID() {
			return traceNodes // 节点已存在，无需添加
		}
	}
	// 节点不存在，添加到列表
	return append(traceNodes, node)
}
