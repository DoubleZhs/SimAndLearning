package recorder

import (
	"sync"
)

var (
	traceDataCache [][][]string = make([][][]string, 0)
	traceDataMutex sync.Mutex   = sync.Mutex{}
)

// func RecordTraceData(vehicle *element.Vehicle) {
// 	traceDataMutex.Lock()
// 	defer traceDataMutex.Unlock()
// 	traceDataCache = append(traceDataCache, getTraceData(vehicle))
// }

// func getTraceData(vehicle *element.Vehicle) [][]string {
// 	trace := vehicle.Trace()
// 	var records [][]string
// 	for _, point := range trace {
// 		vehicleId, time, posId, speed := point.Report()
// 		record := []string{
// 			strconv.FormatInt(vehicleId, 10), // 车辆 ID
// 			strconv.Itoa(time),               // 时间
// 			strconv.FormatInt(posId, 10),     // 节点 ID
// 			strconv.Itoa(speed),              // 速度
// 		}
// 		records = append(records, record)
// 	}
// 	return records
// }

func InitTraceDataCSV(filename string) {
	header := []string{
		"VehicleID", "Time", "PosID", "Speed", "Tag", "ClosedVehicle", "PathLength", "Origin", "Destination",
	}
	initializeCSV(filename, header)
}

func WriteToTraceDataCSV(filename string) {
	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()
	if len(traceDataCache) == 0 {
		return
	}
	for _, records := range traceDataCache {
		appendToCSV(filename, records)
	}
	traceDataCache = make([][][]string, 0)
}
