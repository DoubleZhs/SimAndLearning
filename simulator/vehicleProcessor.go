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
		// 记录轨迹数据 - 确保终点轨迹被记录
		RecordVehicleEndTrace(vehicle, simTime)

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

			// 处理之前行程的轨迹数据
			trace := vehicle.Trace()
			if len(trace) > 0 {
				// 格式化之前行程的轨迹数据并保存
				oldRecordData := recorder.FormatTraceForNewJourney(vehicle)
				if len(oldRecordData) > 0 {
					recorder.SaveTraceData(oldRecordData)
				}
			}

			// 创建新车辆并设置起终点
			newVehicle := element.NewVehicle(id, velocity, acceleration, 1.0, slowingProb, true)
			// 确保轨迹数据为空
			newVehicle.ClearTrace()

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

			// 设置车辆轨迹记录
			// 检查是否启用轨迹记录
			cfg := config.GetConfig()
			traceEnabled := cfg != nil && cfg.Trace.Enabled
			if traceEnabled {
				SetupVehicleTrace(newVehicle, 0) // 使用默认间隔
			}

			// 添加到等待队列
			waitingVehiclesMutex.Lock()
			waitingVehicles[newVehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()
			atomic.AddInt64(&numVehiclesWaiting, 1)
		}

		// 从完成列表中移除车辆
		completedVehiclesMutex.Lock()
		delete(completedVehicles, vehicle)
		completedVehiclesMutex.Unlock()
	}
}

// updateVehicleActiveStatus 更新车辆的激活状态
// 激活状态决定车辆是否能够从缓冲区进入系统
func updateVehicleActiveStatus(numWorkers int) {
	if len(waitingVehicles) == 0 {
		return
	}

	// 使用读写锁以允许并发读取
	waitingVehiclesMutex.RLock()
	// 创建临时列表以避免在迭代过程中修改原映射
	vehiclesToProcess := make([]*element.Vehicle, 0, len(waitingVehicles))
	for vehicle := range waitingVehicles {
		vehiclesToProcess = append(vehiclesToProcess, vehicle)
	}
	waitingVehiclesMutex.RUnlock()

	// 创建记录激活状态的映射
	var recordActivatedVehicle = make(map[*element.Vehicle]struct{})
	var recordMutex sync.Mutex // 添加互斥锁保护map写入

	// 并行处理所有等待中的车辆
	var wg sync.WaitGroup
	wg.Add(len(vehiclesToProcess))

	// 创建工作通道
	vehicleChan := make(chan *element.Vehicle, numWorkers)

	// 启动工作协程
	for i := 0; i < numWorkers; i++ {
		go func() {
			for vehicle := range vehicleChan {
				// 更新车辆激活状态
				if vehicle.UpdateActiveState() {
					vehicle.SystemIn()
					recordMutex.Lock() // 获取锁
					recordActivatedVehicle[vehicle] = struct{}{}
					recordMutex.Unlock() // 释放锁
				}
				wg.Done()
			}
		}()
	}

	// 分发任务
	for _, vehicle := range vehiclesToProcess {
		vehicleChan <- vehicle
	}

	// 关闭通道并等待所有任务完成
	close(vehicleChan)
	wg.Wait()

	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	traceEnabled := cfg != nil && cfg.Trace.Enabled

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

		// 仅在启用轨迹记录时设置轨迹记录
		if traceEnabled {
			// 获取配置的间隔，如果有效则使用它，否则使用默认值
			var interval int
			if cfg != nil {
				interval = cfg.Trace.CheckpointInterval
			}
			SetupVehicleTrace(vehicle, interval)
		}
	}
}

// updateVehiclePosition 更新活动车辆的位置
// 使用工作线程池并发处理所有活动车辆的移动
func updateVehiclePosition(numWorkers, simTime int) {
	if len(activeVehicles) == 0 {
		return
	}

	// 使用读写锁以允许并发读取
	activeVehiclesMutex.RLock()
	// 创建临时列表以避免在迭代过程中修改原映射
	vehiclesToProcess := make([]*element.Vehicle, 0, len(activeVehicles))
	for vehicle := range activeVehicles {
		vehiclesToProcess = append(vehiclesToProcess, vehicle)
	}
	activeVehiclesMutex.RUnlock()

	// 如果没有活动车辆，直接返回
	if len(vehiclesToProcess) == 0 {
		return
	}

	// 创建完成车辆通道
	completedVehicleChan := make(chan *element.Vehicle, len(vehiclesToProcess))

	// 并行处理所有活动车辆
	var wg sync.WaitGroup
	wg.Add(len(vehiclesToProcess))

	// 创建工作通道
	vehicleChan := make(chan *element.Vehicle, numWorkers)

	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	traceEnabled := cfg != nil && cfg.Trace.Enabled

	// 启动工作协程
	for i := 0; i < numWorkers; i++ {
		go func() {
			for vehicle := range vehicleChan {
				// 移动车辆
				if completed := vehicle.Move(simTime); completed {
					// 如果车辆到达终点，将其加入完成通道
					completedVehicleChan <- vehicle
				} else if traceEnabled {
					// 记录轨迹（只在车辆移动后但未完成时记录）
					RecordVehicleTrace(vehicle, simTime)
				}
				wg.Done()
			}
		}()
	}

	// 分发任务
	for _, vehicle := range vehiclesToProcess {
		vehicleChan <- vehicle
	}

	// 关闭通道并等待所有任务完成
	close(vehicleChan)
	wg.Wait()

	// 关闭完成车辆通道
	close(completedVehicleChan)

	// 处理完成的车辆
	for vehicle := range completedVehicleChan {
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

		// 轨迹记录已在Move方法中完成，无需再次处理
	}
}
