package simulator

import (
	"graphCA/config"
	"graphCA/element"
	"graphCA/recorder"
	"sync"
)

var (
	// 存储每个车辆的检查点
	vehicleCheckpoints     = make(map[int64][]int64)
	vehicleCheckpointMutex sync.RWMutex
)

// 获取配置的检查点数量，如果未配置则使用默认值
// 返回的值表示每辆车应该记录的位置点数量
func GetDefaultCheckpointInterval() int {
	cfg := config.GetConfig()
	if cfg != nil && cfg.Trace.CheckpointInterval > 0 {
		return cfg.Trace.CheckpointInterval
	}
	return 10 // 默认值，每辆车记录10个点，系统会根据路径长度调整
}

// SetupVehicleCheckpoints 为车辆设置检查点
// 根据路径长度的百分比设置检查点，而不是固定间隔
// 如果interval为0或负数，则使用默认配置
func SetupVehicleCheckpoints(vehicle *element.Vehicle, interval int) {
	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return // 如果未启用轨迹记录，则不设置检查点
	}

	// 获取车辆ID和路径
	vehicleID := vehicle.Index()
	path := vehicle.GetPath()
	if len(path) == 0 {
		return // 如果路径为空，则不设置检查点
	}

	// 创建检查点列表
	// 固定记录点数为8-12个(包括起点和终点)
	maxCheckpoints := 10 // 平均每辆车的记录点数
	pathLength := len(path)

	checkpoints := make([]int64, 0, maxCheckpoints)

	// 始终添加起点
	checkpoints = append(checkpoints, path[0].ID())

	// 根据路径长度计算应该设置的检查点数量
	var numCheckpoints int
	if pathLength <= 20 { // 对于短路径
		numCheckpoints = 4 // 较少检查点
	} else if pathLength <= 100 {
		numCheckpoints = 6 // 中等数量检查点
	} else {
		numCheckpoints = 8 // 较长路径使用更多检查点
	}

	// 如果用户指定了interval值且大于0，可以用来调整检查点数量
	if interval > 0 {
		// 使用interval作为最大检查点数的控制因子
		numCheckpoints = interval
	}

	// 计算固定百分比位置的检查点
	for i := 1; i < numCheckpoints-1; i++ {
		percentage := float64(i) / float64(numCheckpoints-1)
		index := int(float64(pathLength-1) * percentage)
		if index > 0 && index < pathLength { // 避免起点和终点
			checkpoints = append(checkpoints, path[index].ID())
		}
	}

	// 始终添加终点
	if pathLength > 1 {
		endpointID := path[pathLength-1].ID()
		if checkpoints[len(checkpoints)-1] != endpointID {
			checkpoints = append(checkpoints, endpointID)
		}
	}

	// 保存检查点
	vehicleCheckpointMutex.Lock()
	vehicleCheckpoints[vehicleID] = checkpoints
	vehicleCheckpointMutex.Unlock()
}

// CheckVehicleCheckpoints 检查车辆是否经过了检查点
// 如果车辆经过检查点，则记录轨迹数据
//
// 参数:
//   - vehicle: 要检查的车辆
//   - currentTime: 当前模拟时间
func CheckVehicleCheckpoints(vehicle *element.Vehicle, currentTime int) {
	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return // 如果未启用轨迹记录，则不处理检查点
	}

	if vehicle == nil {
		return
	}

	vehicleID := vehicle.Index()

	// 获取车辆当前位置
	currentPos := vehicle.CurrentPosition()
	if currentPos == nil {
		return
	}

	currentPosID := currentPos.ID()

	// 检查是否是检查点
	vehicleCheckpointMutex.RLock()
	checkpoints, exists := vehicleCheckpoints[vehicleID]
	vehicleCheckpointMutex.RUnlock()

	if !exists {
		return
	}

	// 检查当前位置是否是检查点
	isCheckpoint := false
	for _, checkpoint := range checkpoints {
		if checkpoint == currentPosID {
			isCheckpoint = true
			break
		}
	}

	if isCheckpoint {
		// 在当前位置添加轨迹点
		vehicle.AddTracePoint(currentPosID, currentTime)

		// 记录轨迹数据
		recorder.RecordTraceData(vehicle)
	}
}

// ClearVehicleCheckpoints 清除车辆的检查点
// 当车辆完成行程时调用此函数
//
// 参数:
//   - vehicleID: 要清除检查点的车辆ID
func ClearVehicleCheckpoints(vehicleID int64) {
	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return // 如果未启用轨迹记录，则不处理检查点
	}

	vehicleCheckpointMutex.Lock()
	delete(vehicleCheckpoints, vehicleID)
	vehicleCheckpointMutex.Unlock()
}
