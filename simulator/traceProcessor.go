package simulator

import (
	"graphCA/config"
	"graphCA/element"
	"graphCA/recorder"

	"gonum.org/v1/gonum/graph"
)

// SetupVehicleTrace 为车辆设置轨迹记录间隔
// 如果interval为0或负数，则使用默认配置
func SetupVehicleTrace(vehicle *element.Vehicle, interval int) {
	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return // 如果未启用轨迹记录，则不设置
	}

	// 如果未指定间隔或间隔无效，使用配置的间隔
	if interval <= 0 {
		if cfg.Trace.CheckpointInterval > 0 {
			interval = cfg.Trace.CheckpointInterval
		} else {
			interval = 10 // 默认值，每10个时间步记录一次
		}
	}

	// 设置车辆的轨迹记录间隔
	vehicle.SetTraceInterval(interval)

	// 车辆状态≥3表示已进入系统（包括缓冲区），应该开始记录轨迹
	// state=3: 进入缓冲区 - 车辆已进入系统，需要开始记录轨迹
	// state=4: 在路网上行驶
	vehicleState := vehicle.State()
	if vehicleState >= 3 {
		// 记录起点
		if origin := vehicle.Origin(); origin != nil {
			// 如果车辆在缓冲区，记录起点和进入时间
			// 如果车辆已在路网，记录当前位置
			if vehicleState == 3 {
				vehicle.AddTracePoint(origin.ID(), vehicle.InTime())
			} else { // state = 4
				vehicle.AddTracePoint(vehicle.CurrentPosition().ID(), vehicle.InTime())
			}
		}
	}
}

// RecordVehicleTrace 根据条件记录车辆轨迹
// 如果车辆满足记录条件，将添加当前位置到轨迹并记录数据
func RecordVehicleTrace(vehicle *element.Vehicle, currentTime int) {
	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return // 如果未启用轨迹记录，则不处理
	}

	if vehicle == nil {
		return
	}

	// 只有当车辆已进入系统(state≥3)时才记录轨迹
	vehicleState := vehicle.State()
	if vehicleState < 3 || vehicleState > 4 {
		return // 只记录state=3(缓冲区)和state=4(路网中)的车辆
	}

	// 判断是否应该记录轨迹
	if vehicle.ShouldRecordTrace(currentTime) {
		// 获取车辆当前位置
		var positionNode graph.Node
		if vehicleState == 3 {
			// 如果车辆在缓冲区，记录起点
			positionNode = vehicle.Origin()
		} else { // state = 4
			// 如果车辆在路网中，记录当前位置
			positionNode = vehicle.CurrentPosition()
		}

		if positionNode == nil {
			return
		}

		// 添加轨迹点
		vehicle.AddTracePoint(positionNode.ID(), currentTime)

		// 记录轨迹数据
		recorder.RecordTraceData(vehicle)
	}
}

// 当车辆完成旅程时，记录终点轨迹
func RecordVehicleEndTrace(vehicle *element.Vehicle, currentTime int) {
	// 检查是否启用轨迹记录
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return // 如果未启用轨迹记录，则不处理
	}

	if vehicle == nil {
		return
	}

	// 记录目的地作为最后一个轨迹点
	if destination := vehicle.Destination(); destination != nil {
		vehicle.AddTracePoint(destination.ID(), currentTime)
		recorder.RecordTraceData(vehicle)
	}
}
