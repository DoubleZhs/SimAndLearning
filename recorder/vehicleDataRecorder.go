package recorder

import (
	"fmt"
	"simAndLearning/element"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"gonum.org/v1/gonum/graph"
)

var (
	vehicleDataCache [][]string = make([][]string, 0)
	vehicleDataMutex sync.Mutex = sync.Mutex{}
	recordIndex      int64      = 0 // 递增的唯一索引
)

func RecordVehicleData(vehicle *element.Vehicle) {
	vehicleDataMutex.Lock()
	defer vehicleDataMutex.Unlock()
	vehicleDataCache = append(vehicleDataCache, getVehicleData(vehicle))
}

func getVehicleData(vehicle *element.Vehicle) []string {
	// 生成唯一递增索引
	idx := atomic.AddInt64(&recordIndex, 1)

	// 基本信息
	index, acceleration, slowingProb, inTime, outTime, tag, flag := vehicle.Report()
	// 起终点ID
	originId := vehicle.Origin().ID()
	destinationId := vehicle.Destination().ID()
	pathlength := vehicle.PathLength()

	// 获取路径
	simplePath := formatSimplePath(vehicle.GetPath())

	return []string{
		strconv.FormatInt(idx, 10),           // 新增的唯一索引
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
		simplePath,                           // 车辆路径
	}
}

// formatSimplePath 将车辆路径格式化为字符串
func formatSimplePath(path []graph.Node) string {
	if path == nil || len(path) == 0 {
		return "[]"
	}

	nodeIds := make([]string, len(path))
	for i, node := range path {
		nodeIds[i] = strconv.FormatInt(node.ID(), 10)
	}

	return "[" + strings.Join(nodeIds, ",") + "]"
}

func InitVehicleDataCSV(filename string) {
	header := []string{
		"Trip ID", "Vehicle ID", "Acceleration", "SlowingPro", "Origin", "Destination", "In Time", "Arrival Time", "Tag", "ClosedVehicle", "PathLength", "Path",
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
