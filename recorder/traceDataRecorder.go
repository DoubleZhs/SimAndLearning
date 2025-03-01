package recorder

import (
	"graphCA/element"
	"strconv"
	"sync"
	"time"

	"gonum.org/v1/gonum/graph"
)

var (
	traceDataCache [][][]string = make([][][]string, 0, 1000) // 预分配更大的容量
	traceDataMutex sync.RWMutex = sync.RWMutex{}              // 使用读写锁提高并发性能
)

// TracePoint 表示轨迹中的一个记录点
type TracePoint struct {
	VehicleID     int64
	Time          int
	PositionID    int64
	Speed         int
	Acceleration  int
	Tag           string // 标记类型，如"checkpoint"、"periodic"等
	ClosedVehicle bool
	PathLength    int
	Origin        int64
	Destination   int64
}

// RecordTraceData 记录单个车辆的轨迹数据
// 会将车辆的轨迹信息转换为CSV格式的记录并存入缓存
func RecordTraceData(vehicle *element.Vehicle) {
	if vehicle == nil {
		return
	}

	traceData := getTraceData(vehicle, "event")

	// 使用读写锁的写锁来保护写操作
	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	traceDataCache = append(traceDataCache, traceData)
}

// getTraceData 从车辆获取轨迹数据并格式化为CSV记录
// tag参数用于标记数据来源（如"periodic"、"checkpoint"等）
func getTraceData(vehicle *element.Vehicle, tag string) [][]string {
	trace := vehicle.Trace()
	vehicleID := vehicle.Index()
	isClosed := vehicle.Flag()
	pathLength := vehicle.PathLength()

	origin, destination := int64(-1), int64(-1)
	if od := vehicle.GetOD(); od != nil && len(od) >= 2 {
		origin = od[0].ID()
		destination = od[1].ID()
	}

	records := make([][]string, 0, len(trace))
	for posID, timeStamp := range trace {
		speed := vehicle.Velocity()
		acceleration := vehicle.Acceleration()

		record := []string{
			strconv.FormatInt(vehicleID, 10),   // 车辆 ID
			strconv.Itoa(timeStamp),            // 时间
			strconv.FormatInt(posID, 10),       // 节点 ID
			strconv.Itoa(speed),                // 速度
			tag,                                // 标记类型
			strconv.FormatBool(isClosed),       // 是否为循环车辆
			strconv.Itoa(pathLength),           // 路径长度
			strconv.FormatInt(origin, 10),      // 起点
			strconv.FormatInt(destination, 10), // 终点
			strconv.Itoa(acceleration),         // 加速度
		}
		records = append(records, record)
	}
	return records
}

// PeriodicTraceRecording 定期记录所有活跃车辆的轨迹
// interval: 记录间隔时间（单位：模拟时间步长）
// nodes: 网络中的所有节点
// stopChan: 用于停止周期性记录的通道
func PeriodicTraceRecording(interval int, nodes []graph.Node, stopChan <-chan struct{}) {
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	simTime := 0

	for {
		select {
		case <-ticker.C:
			simTime += interval
			// 扫描网络中的所有节点收集车辆信息
			ScanNetworkForVehicles(simTime, nodes, "periodic")
		case <-stopChan:
			return
		}
	}
}

// ScanNetworkForVehicles 扫描网络中的所有节点，记录活跃车辆信息
// simTime: 当前模拟时间
// nodes: 网络中的所有节点
// tag: 记录类型标签
func ScanNetworkForVehicles(simTime int, nodes []graph.Node, tag string) {
	vehiclesRecorded := make(map[int64]struct{})

	for _, node := range nodes {
		cell, ok := node.(element.Cell)
		if !ok {
			continue
		}

		vehicles := cell.ListContainer()
		for _, vehicle := range vehicles {
			// 避免重复记录同一车辆
			if _, exists := vehiclesRecorded[vehicle.Index()]; exists {
				continue
			}

			// 将当前位置添加到车辆轨迹（注意：这可能需要在vehicle.go中实现相应方法）
			// vehicle.AddTracePoint(cell.ID(), simTime)

			// 记录车辆数据
			traceData := getTraceData(vehicle, tag)

			traceDataMutex.Lock()
			traceDataCache = append(traceDataCache, traceData)
			traceDataMutex.Unlock()

			vehiclesRecorded[vehicle.Index()] = struct{}{}
		}
	}
}

// SetupCheckpoints 在车辆路径上设置检查点
// vehicle: 车辆对象
// checkpointInterval: 检查点间隔（节点数）
// 返回检查点节点ID列表
func SetupCheckpoints(vehicle *element.Vehicle, checkpointInterval int) []int64 {
	// 获取车辆路径
	path := vehicle.GetPath()
	if path == nil || len(path) == 0 {
		return nil
	}

	checkpoints := make([]int64, 0, len(path)/checkpointInterval+2)

	// 确保起点和终点始终是检查点
	checkpoints = append(checkpoints, path[0].ID())

	// 添加中间检查点
	for i := checkpointInterval; i < len(path); i += checkpointInterval {
		checkpoints = append(checkpoints, path[i].ID())
	}

	// 添加终点（如果不是已添加的检查点）
	endpointID := path[len(path)-1].ID()
	if len(checkpoints) == 0 || checkpoints[len(checkpoints)-1] != endpointID {
		checkpoints = append(checkpoints, endpointID)
	}

	return checkpoints
}

// InitTraceDataCSV 初始化轨迹数据CSV文件
func InitTraceDataCSV(filename string) {
	header := []string{
		"VehicleID", "Time", "PosID", "Speed", "Tag", "ClosedVehicle", "PathLength", "Origin", "Destination", "Acceleration",
	}
	initializeCSV(filename, header)
}

// WriteToTraceDataCSV 将缓存的轨迹数据写入CSV文件
func WriteToTraceDataCSV(filename string) {
	// 使用读锁，允许并发读取，但阻止写入
	traceDataMutex.RLock()

	// 复制数据以尽快释放锁
	var dataToWrite [][][]string
	if len(traceDataCache) > 0 {
		dataToWrite = make([][][]string, len(traceDataCache))
		copy(dataToWrite, traceDataCache)
	}

	traceDataMutex.RUnlock()

	// 如果没有数据，直接返回
	if len(dataToWrite) == 0 {
		return
	}

	// 写入数据到CSV
	for _, records := range dataToWrite {
		appendToCSV(filename, records)
	}

	// 清空缓存
	traceDataMutex.Lock()
	// 只有当缓存没有新增数据时才清空
	if len(traceDataCache) == len(dataToWrite) {
		traceDataCache = make([][][]string, 0, 1000)
	} else {
		// 如果有新数据，只删除已写入的部分
		traceDataCache = traceDataCache[len(dataToWrite):]
	}
	traceDataMutex.Unlock()
}
