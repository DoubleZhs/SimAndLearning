package recorder

import (
	"fmt"
	"os"
	"simAndLearning/element"
	"strconv"
	"sync"

	"gonum.org/v1/gonum/graph"
)

var (
	// 按天存储轨迹数据，key为天数，value为轨迹数据
	traceDataCacheByDay map[int][][]string = make(map[int][][]string)
	traceDataMutex      sync.Mutex         = sync.Mutex{}
	oneDayTimeSteps     int                = 57600 // 一天的时间步数
)

// SetOneDayTimeSteps 设置一天的时间步数
func SetOneDayTimeSteps(steps int) {
	if steps > 0 {
		oneDayTimeSteps = steps
	}
}

// getDay 根据时间步获取天数
func getDay(timeStep int) int {
	return timeStep/oneDayTimeSteps + 1
}

// RecordTraceData 记录车辆轨迹数据
func RecordTraceData(vehicleID int64, time int, position graph.Node) {
	day := getDay(time)

	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	// 确保该天的数据切片已初始化
	if _, exists := traceDataCacheByDay[day]; !exists {
		traceDataCacheByDay[day] = make([][]string, 0)
	}

	traceDataCacheByDay[day] = append(traceDataCacheByDay[day], getTraceData(vehicleID, time, position))
}

// RecordVehicleTrace 记录车辆所有轨迹数据
func RecordVehicleTrace(vehicle *element.Vehicle) {
	trace := vehicle.GetTrace()
	if len(trace) == 0 {
		return
	}

	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	vehicleID := vehicle.Index()
	for time, position := range trace {
		day := getDay(time)

		// 确保该天的数据切片已初始化
		if _, exists := traceDataCacheByDay[day]; !exists {
			traceDataCacheByDay[day] = make([][]string, 0)
		}

		traceDataCacheByDay[day] = append(traceDataCacheByDay[day], getTraceData(vehicleID, time, position))
	}
}

// getTraceData 获取轨迹数据格式
func getTraceData(vehicleID int64, time int, position graph.Node) []string {
	return []string{
		strconv.FormatInt(vehicleID, 10),     // 车辆ID
		strconv.Itoa(time),                   // 时间戳
		strconv.FormatInt(position.ID(), 10), // 位置ID
	}
}

// GetDailyTraceDataFilename 获取指定天数的轨迹数据文件名
func GetDailyTraceDataFilename(baseFilename string, day int) string {
	// 从基础文件名提取前缀和后缀
	// 假设格式为 ./data/时间戳_车辆数_TraceData.csv
	filenameSuffix := "_TraceData.csv"
	basePath := baseFilename[:len(baseFilename)-len(filenameSuffix)]

	// 构建轨迹数据目录名称
	dirName := basePath + "_TraceData"

	// 确保目录存在
	ensureDirectoryExists(dirName)

	// 返回目录下的文件路径
	return fmt.Sprintf("%s/Day%d.csv", dirName, day)
}

// ensureDirectoryExists 确保目录存在，不存在则创建
func ensureDirectoryExists(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dirPath, err)
		}
	}
}

// InitTraceDataCSV 初始化轨迹数据目录
// 现在只需要确保目录存在，不再创建整体文件
func InitTraceDataCSV(filename string) {
	// 从基础文件名提取前缀
	filenameSuffix := "_TraceData.csv"
	basePath := filename[:len(filename)-len(filenameSuffix)]

	// 构建轨迹数据目录名称
	dirName := basePath + "_TraceData"

	// 确保目录存在
	ensureDirectoryExists(dirName)
}

// WriteToTraceDataCSV 将缓存的轨迹数据写入CSV文件
// 按天分别写入不同的文件
func WriteToTraceDataCSV(baseFilename string) {
	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	// 遍历所有天的数据
	for day, data := range traceDataCacheByDay {
		if len(data) == 0 {
			continue
		}

		// 获取当天的文件名
		filename := GetDailyTraceDataFilename(baseFilename, day)

		// 如果是首次写入该天的数据，需要初始化CSV文件
		// 检查文件是否存在，不存在则创建并写入表头
		if !fileExists(filename) {
			header := []string{
				"Vehicle ID", "Time", "Position",
			}
			initializeCSV(filename, header)
		}

		// 写入数据
		appendToCSV(filename, data)

		// 清空该天的缓存
		traceDataCacheByDay[day] = make([][]string, 0)
	}
}
