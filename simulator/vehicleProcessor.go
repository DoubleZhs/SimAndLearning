package simulator

import (
	"graphCA/config"
	"graphCA/element"
	"graphCA/recorder"
	"graphCA/utils"
	"sync"
	"sync/atomic"

	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// VehicleProcess 处理当前模拟环境中所有车辆的状态
// 依次执行：检查已完成车辆、更新车辆激活状态、更新车辆位置、处理检查点
func VehicleProcess(numWorkers, simTime int, g *simple.DirectedGraph, traceNodes []graph.Node) {
	checkCompletedVehicle(simTime, g, traceNodes)
	updateVehicleActiveStatus(numWorkers)
	updateVehiclePosition(numWorkers, simTime)
}

// checkCompletedVehicle 处理已完成行程的车辆
// 记录数据并根据车辆类型决定是否重新进入系统
func checkCompletedVehicle(simTime int, g *simple.DirectedGraph, traceNodes []graph.Node) {
	if len(completedVehicles) == 0 {
		return
	}

	// 使用读写锁以允许并发读取
	completedVehiclesMutex.RLock()
	// 创建临时列表以避免在迭代过程中修改原映射
	vehiclesToProcess := make([]*element.Vehicle, 0, len(completedVehicles))
	for vehicle := range completedVehicles {
		vehiclesToProcess = append(vehiclesToProcess, vehicle)
	}
	completedVehiclesMutex.RUnlock()

	for _, vehicle := range vehiclesToProcess {
		// 记录车辆数据
		recorder.RecordVehicleData(vehicle)

		// 仅处理闭环车辆（需要重新进入系统的车辆）
		if vehicle.Flag() {
			id, velocity, acceleration, slowingProb := vehicle.Index(), vehicle.Velocity(), vehicle.Acceleration(), vehicle.SlowingProb()

			// 为车辆选择新的起点和终点
			newO := vehicle.Destination()
			minLength, maxLength := TripDistanceRange()

			// 获取可达节点
			allowedDCells := utils.AccessibleNodesWithinRange(g, newO, minLength, maxLength)
			if len(allowedDCells) == 0 {
				continue // 如果没有可达节点，跳过此车辆
			}

			newD := allowedDCells[rand.IntN(len(allowedDCells))]

			// 更新追踪节点列表
			var findO, findD bool
			for _, node := range traceNodes {
				if node.ID() == newO.ID() {
					findO = true
				}
				if node.ID() == newD.ID() {
					findD = true
				}
			}

			if !findO {
				traceNodes = append(traceNodes, newO)
			}
			if !findD {
				traceNodes = append(traceNodes, newD)
			}

			// 创建新车辆并设置起终点
			newVehicle := element.NewVehicle(id, velocity, acceleration, 1.0, slowingProb, true)
			if ok, err := newVehicle.SetOD(g, newO, newD); !ok {
				if err != nil {
					// 记录错误并跳过此车辆
					continue
				}
			}

			// 设置路径
			path, _, err := utils.ShortestPath(g, newO, newD)
			if err != nil {
				continue // 如果无法找到路径，跳过此车辆
			}

			if ok, err := newVehicle.SetPath(path); !ok {
				if err != nil {
					continue
				}
			}

			// 将车辆放入缓冲区
			newVehicle.BufferIn(simTime)

			// 添加到等待车辆列表
			waitingVehiclesMutex.Lock()
			waitingVehicles[newVehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()

			// 更新等待车辆计数
			atomic.AddInt64(&numVehiclesWaiting, 1)
		}

		// 从完成列表中移除车辆
		completedVehiclesMutex.Lock()
		delete(completedVehicles, vehicle)
		completedVehiclesMutex.Unlock()
	}
}

// updateVehicleActiveStatus 更新车辆的激活状态
// 检查等待中的车辆是否可以激活，并将激活的车辆移动到活动列表
func updateVehicleActiveStatus(numWorkers int) {
	if len(waitingVehicles) == 0 {
		return
	}

	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	traceEnabled := cfg != nil && cfg.Trace.Enabled

	// 获取所有等待中的车辆
	waitingVehiclesMutex.RLock()
	vehicles := make([]*element.Vehicle, 0, len(waitingVehicles))
	for vehicle := range waitingVehicles {
		vehicles = append(vehicles, vehicle)
	}
	waitingVehiclesMutex.RUnlock()

	// 优化工作线程数量
	if len(vehicles) < numWorkers {
		numWorkers = len(vehicles)
	}

	vehiclesPerThread := len(vehicles) / numWorkers
	extraVehicles := len(vehicles) % numWorkers

	var wg sync.WaitGroup
	recordActivatedVehicle := make(chan *element.Vehicle, len(vehicles))

	start := 0
	for i := 0; i < numWorkers; i++ {
		end := start + vehiclesPerThread
		if i < extraVehicles {
			end++
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end && j < len(vehicles); j++ {
				vehicle := vehicles[j]
				ok := vehicle.UpdateActiveState()
				if ok {
					recordActivatedVehicle <- vehicle
				}
			}
		}(start, end)

		start = end
	}

	// 等待所有工作线程完成
	wg.Wait()
	close(recordActivatedVehicle)

	// 处理激活的车辆
	for vehicle := range recordActivatedVehicle {
		// 从等待列表移到活动列表
		waitingVehiclesMutex.Lock()
		delete(waitingVehicles, vehicle)
		waitingVehiclesMutex.Unlock()
		atomic.AddInt64(&numVehiclesWaiting, -1)

		activeVehiclesMutex.Lock()
		activeVehicles[vehicle] = struct{}{}
		activeVehiclesMutex.Unlock()
		atomic.AddInt64(&numVehiclesActive, 1)

		// 仅在启用轨迹记录时设置检查点
		if traceEnabled {
			// 获取检查点间隔，如果配置有效则使用配置值，否则使用默认值
			var checkpointInterval int
			if cfg != nil {
				checkpointInterval = cfg.Trace.CheckpointInterval
			}
			SetupVehicleCheckpoints(vehicle, checkpointInterval)
		}
	}
}

// updateVehiclePosition 更新活动车辆的位置
// 使用工作线程池并发处理所有活动车辆的移动
func updateVehiclePosition(numWorkers, simTime int) {
	if len(activeVehicles) == 0 {
		return
	}

	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	traceEnabled := cfg != nil && cfg.Trace.Enabled

	// 获取所有活动车辆并随机打乱顺序
	activeVehiclesMutex.RLock()
	vehicles := make([]*element.Vehicle, 0, len(activeVehicles))
	for vehicle := range activeVehicles {
		vehicles = append(vehicles, vehicle)
	}
	activeVehiclesMutex.RUnlock()

	// 随机打乱车辆顺序以避免系统性偏差
	for i := len(vehicles) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		vehicles[i], vehicles[j] = vehicles[j], vehicles[i]
	}

	// 优化工作线程数量
	if len(vehicles) < numWorkers {
		numWorkers = len(vehicles)
	}

	vehiclesPerThread := len(vehicles) / numWorkers
	extraVehicles := len(vehicles) % numWorkers

	// 使用channel收集已完成的车辆
	completedVehicleChan := make(chan *element.Vehicle, len(vehicles))

	var wg sync.WaitGroup

	start := 0
	for i := 0; i < numWorkers; i++ {
		end := start + vehiclesPerThread
		if i < extraVehicles {
			end++
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end && j < len(vehicles); j++ {
				vehicle := vehicles[j]
				if vehicle.State() == 3 {
					vehicle.SystemIn()
				}
				if vehicle.State() == 4 {
					completed := vehicle.Move(simTime)

					// 如果启用了轨迹记录，检查车辆是否经过检查点
					if traceEnabled {
						CheckVehicleCheckpoints(vehicle, simTime)
					}

					if completed {
						// 当车辆完成移动后，将其加入到完成通道
						completedVehicleChan <- vehicle
					}
				}
			}
		}(start, end)

		start = end
	}

	// 在主协程中等待所有工作协程完成
	wg.Wait()
	close(completedVehicleChan)

	// 在主协程中处理已完成的车辆
	for vehicle := range completedVehicleChan {
		// 获取车辆ID用于清除检查点
		vehicleID := vehicle.Index()

		// 执行SystemOut操作，确保车辆被正确卸载
		vehicle.SystemOut(simTime)

		// 更新各种状态
		activeVehiclesMutex.Lock()
		delete(activeVehicles, vehicle)
		activeVehiclesMutex.Unlock()
		atomic.AddInt64(&numVehiclesActive, -1)

		completedVehiclesMutex.Lock()
		completedVehicles[vehicle] = struct{}{}
		completedVehiclesMutex.Unlock()
		atomic.AddInt64(&numVehicleCompleted, 1)

		// 仅在启用轨迹记录时清除检查点
		if traceEnabled {
			ClearVehicleCheckpoints(vehicleID)
		}
	}
}
