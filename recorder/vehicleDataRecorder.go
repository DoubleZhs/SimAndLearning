package recorder

import (
	"fmt"
	"graphCA/element"
	"strconv"
	"sync"
)

var (
	vehicleDataCache [][]string = make([][]string, 0)
	vehicleDataMutex sync.Mutex = sync.Mutex{}
)

func RecordVehicleData(vehicle *element.Vehicle) {
	vehicleDataMutex.Lock()
	defer vehicleDataMutex.Unlock()
	vehicleDataCache = append(vehicleDataCache, getVehicleData(vehicle))
}

func getVehicleData(vehicle *element.Vehicle) []string {
	// 基本信息
	index, acceleration, slowingProb, inTime, outTime, tag, flag := vehicle.Report()
	// 起终点ID
	originId := vehicle.Origin().ID()
	destinationId := vehicle.Destination().ID()
	pathlength := vehicle.PathLength()

	return []string{
		strconv.FormatInt(index, 10),         // 车辆 ID
		strconv.Itoa(acceleration),           // 车辆加速度
		fmt.Sprintf("%.4f", slowingProb),     // 减速概率
		strconv.FormatInt(originId, 10),      // 起点 ID
		strconv.FormatInt(destinationId, 10), // 终点 ID
		strconv.Itoa(inTime),                 // 进入系统时间
		strconv.Itoa(outTime),                // 到达时间
		fmt.Sprintf("%.4f", tag),             // 标签
		strconv.FormatBool(flag),             // 是否为封闭系统车辆
		strconv.Itoa(pathlength),             // 路径长度（元胞数）
	}
}

func InitVehicleDataCSV(filename string) {
	header := []string{
		"Vehicle ID", "Acceleration", "SlowingPro", "Origin", "Destination", "In Time", "Arrival Time", "Tag", "ClosedVehicle", "PathLength",
	}
	initializeCSV(filename, header)
}

func WriteToVehicleDataCSV(filename string) {
	vehicleDataMutex.Lock()
	defer vehicleDataMutex.Unlock()
	if len(vehicleDataCache) == 0 {
		return
	}
	appendToCSV(filename, vehicleDataCache)
	vehicleDataCache = make([][]string, 0)
}
