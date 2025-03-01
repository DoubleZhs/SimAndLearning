package recorder

import (
	"fmt"
	"strconv"
	"sync"
)

var (
	systemDataCache [][]string = make([][]string, 0)
	systemDataMutex sync.Mutex = sync.Mutex{}
)

func RecordSystemData(simTime int, numVehicleGenerated, numVehiclesActive, numVehiclesWaiting, numVehicleCompleted int64, averageSpeed, vehicleDensity float64) {
	systemDataMutex.Lock()
	defer systemDataMutex.Unlock()
	systemDataCache = append(systemDataCache, formatSystemState(simTime, numVehicleGenerated, numVehiclesActive, numVehiclesWaiting, numVehicleCompleted, averageSpeed, vehicleDensity))
}

func formatSystemState(simTime int, numVehicleGenerated, numVehiclesActive, numVehiclesWaiting, numVehicleCompleted int64, averageSpeed, vehicleDensity float64) []string {
	timeOfDay := simTime % 57600
	day := simTime/57600 + 1
	return []string{
		strconv.Itoa(simTime),
		strconv.Itoa(day),                          // 当前天数
		strconv.Itoa(timeOfDay),                    // 时间步
		strconv.FormatInt(numVehicleGenerated, 10), // 道路车辆数量
		strconv.FormatInt(numVehiclesActive, 10),   // 道路车辆数量
		strconv.FormatInt(numVehiclesWaiting, 10),  // 道路车辆数量
		strconv.FormatInt(numVehicleCompleted, 10), // 道路车辆数量
		fmt.Sprintf("%.4f", averageSpeed),          // 平均速度
		fmt.Sprintf("%.4f", vehicleDensity),        // 车辆密度
	}
}

func InitSystemDataCSV(filename string) {
	header := []string{
		"SimTime", "Day", "TimeOfDay", "NumVehicleGenerated", "NumVehiclesActive", "NumVehiclesWaiting", "NumVehicleCompleted", "AverageSpeed", "VehicleDensity",
	}
	initializeCSV(filename, header)
}

func WriteToSystemDataCSV(filename string) {
	systemDataMutex.Lock()
	defer systemDataMutex.Unlock()
	if len(systemDataCache) == 0 {
		return
	}
	appendToCSV(filename, systemDataCache)
	systemDataCache = make([][]string, 0)
}
