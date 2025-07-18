package simulator

import (
	"simAndLearning/element"
	"simAndLearning/recorder"
	"simAndLearning/utils"
	"sync"
	"sync/atomic"

	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// VehicleProcess 处理当前模拟环境中所有车辆的状态
// 依次执行：检查已完成车辆、更新车辆激活状态、更新车辆位置、处理检查点
func VehicleProcess(numWorkers, simTime int, g *simple.DirectedGraph) {
	checkCompletedVehicle(simTime, g)
	updateVehicleActiveStatus(numWorkers)
	updateVehiclePosition(numWorkers, simTime)
}

// checkCompletedVehicle 处理已完成行程的车辆
// 记录数据并根据车辆类型决定是否重新进入系统
func checkCompletedVehicle(simTime int, g *simple.DirectedGraph) {
	if len(completedVehicles) == 0 {
		return
	}

	// 获取配置的路径查找器
	pathFinder := utils.GetPathFinder()

	for vehicle := range completedVehicles {
		// 记录车辆数据
		recorder.RecordVehicleData(vehicle)
		// 记录车辆轨迹数据
		recorder.RecordVehicleTrace(vehicle)

		// 仅处理闭环车辆（需要重新进入系统的车辆）
		if vehicle.Flag() {
			// 为车辆选择新的起点和终点
			newO := vehicle.Destination()

			// 根据是否启用距离限制选择不同的方式获取终点
			var newD graph.Node
			if isDistanceLimitEnabled() {
				minLength, maxLength := TripDistanceRange()

				// 获取可达节点
				allowedDCells := utils.AccessibleNodesWithinRange(g, newO, minLength, maxLength)
				if len(allowedDCells) == 0 {
					continue // 如果没有可达节点，跳过此车辆
				}

				newD = allowedDCells[rand.IntN(len(allowedDCells))]
			} else {
				// 即使不启用距离限制，也确保最小距离在1英里以上
				minLength, _ := TripDistanceRange() // 使用TripDistanceRange获取最小距离，已确保大于1英里
				allowedDCells := utils.AccessibleNodesWithinRange(g, newO, minLength, 1000000)
				if len(allowedDCells) == 0 {
					continue // 如果没有合适的终点，跳过此车辆
				}

				// 从可达节点中随机选择一个作为终点
				newD = allowedDCells[rand.IntN(len(allowedDCells))]
			}

			// 保留原车辆的ID和属性，重新设置起点和终点
			// 获取原车辆的各项属性
			vehicleID := vehicle.Index()
			vehicleVelocity := vehicle.Velocity()
			vehicleAcceleration := vehicle.Acceleration()
			vehicleOccupy := vehicle.Occupy() // 使用Occupy方法获取原车辆的占用空间
			vehicleSlowingProb := vehicle.SlowingProb()

			// 创建新车辆，保持原有属性
			newVehicle := element.NewVehicle(
				vehicleID,           // 保持原车辆ID
				vehicleVelocity,     // 保持原车辆速度
				vehicleAcceleration, // 保持原车辆加速度
				vehicleOccupy,       // 保持原车辆占用空间
				vehicleSlowingProb,  // 保持原车辆减速概率
				true,                // 保持为闭环车辆(flag=true)
			)

			if ok, err := newVehicle.SetOD(g, newO, newD); !ok {
				if err != nil {
					// 记录错误并跳过此车辆
					continue
				}
			}

			// 设置路径（使用配置的路径查找方法）
			path, _, err := pathFinder(g, newO, newD)
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

	// 启动工作协程
	for i := 0; i < numWorkers; i++ {
		go func() {
			for vehicle := range vehicleChan {
				// 移动车辆
				completed := vehicle.Move(simTime)

				// 如果车辆到达终点，将其加入完成通道
				if completed {
					completedVehicleChan <- vehicle
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
		// 更新各种状态
		activeVehiclesMutex.Lock()
		delete(activeVehicles, vehicle)
		activeVehiclesMutex.Unlock()
		atomic.AddInt64(&numVehiclesActive, -1)

		completedVehiclesMutex.Lock()
		completedVehicles[vehicle] = struct{}{}
		completedVehiclesMutex.Unlock()
		atomic.AddInt64(&numVehicleCompleted, 1)
	}
}
