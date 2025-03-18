package recorder

import (
	"fmt"
	"graphCA/config"
	"graphCA/element"
	"graphCA/log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	traceDataCache [][][]string // 轨迹数据缓存
	traceDataMutex sync.RWMutex // 使用读写锁保护并发访问

	// 内存监控相关变量
	recordCount int32 = 0 // 记录计数器，用于定期检查内存占用
	isWriting   int32 = 0 // 写入状态标志，避免并发写入

	// 配置参数，会在初始化时从config包读取
	maxCacheSize        int  // 最大缓存条目数，仅用于初始化时的容量预分配，不再用于触发写入
	memoryThresholdMB   int  // 内存阈值(MB)，超过此值触发写入
	memoryCheckInterval int  // 内存检查间隔(记录次数)
	enableMemMonitor    bool // 是否启用内存监控
)

// 初始化函数，从配置加载内存管理参数
func init() {
	// 初始时使用默认值，配置加载后会被更新
	traceDataCache = make([][][]string, 0, 1000)
	maxCacheSize = 100000     // 仅用于预分配空间
	memoryThresholdMB = 500   // 默认阈值500MB
	memoryCheckInterval = 500 // 增加检查频率，提前发现内存增长
	enableMemMonitor = true
}

// 更新配置参数，从config包读取最新配置
func UpdateConfig() {
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Trace.Enabled {
		return
	}

	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	maxCacheSize = cfg.Trace.MaxCacheSize // 仅用于控制预分配空间大小
	memoryThresholdMB = cfg.Trace.MemoryThreshold
	memoryCheckInterval = cfg.Trace.MemoryCheckInterval
	enableMemMonitor = cfg.Trace.EnableMemoryMonitor

	// 如果缓存容量不足，进行扩容 - 保留这个逻辑，但仅用于预分配空间
	if cap(traceDataCache) < maxCacheSize && len(traceDataCache) < maxCacheSize/2 {
		newCapacity := max(cap(traceDataCache)*2, maxCacheSize)
		newCache := make([][][]string, len(traceDataCache), newCapacity)
		copy(newCache, traceDataCache)
		traceDataCache = newCache
	}

	log.WriteLog(fmt.Sprintf("Trace recorder config updated: memThreshold=%dMB, checkInterval=%d, memMonitor=%v, cacheCapacity=%d",
		memoryThresholdMB, memoryCheckInterval, enableMemMonitor, cap(traceDataCache)))
}

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

	traceData := getTraceData(vehicle, "trace")
	if len(traceData) == 0 {
		return // 如果没有轨迹数据，直接返回
	}

	// 使用读写锁的写锁来保护写操作
	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	traceDataCache = append(traceDataCache, traceData)

	// 增加记录计数
	count := atomic.AddInt32(&recordCount, 1)

	// 周期性检查内存占用 - 更频繁检查，根据记录数量和配置的检查间隔
	if enableMemMonitor && count%int32(memoryCheckInterval) == 0 {
		go checkMemoryUsage()
	}
}

// checkMemoryUsage 检查内存使用情况，如果超过阈值，触发写入
func checkMemoryUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memUsageMB := int(memStats.Alloc / (1024 * 1024))

	// 如果内存使用超过阈值的80%，开始记录警告
	if memUsageMB > int(float64(memoryThresholdMB)*0.8) {
		log.WriteLog(fmt.Sprintf("Memory usage approaching threshold: %dMB/%dMB (%.1f%%)",
			memUsageMB, memoryThresholdMB, float64(memUsageMB)/float64(memoryThresholdMB)*100))
	}

	// 仅在超过阈值时触发写入
	if memUsageMB > memoryThresholdMB {
		log.WriteLog(fmt.Sprintf("Memory threshold reached: %dMB/%dMB. Triggering trace write operation.",
			memUsageMB, memoryThresholdMB))
		triggerTraceWrite()
	}
}

// triggerTraceWrite 触发轨迹数据写入，避免重复触发
func triggerTraceWrite() {
	// 如果已经有写入操作在进行，直接返回
	if !atomic.CompareAndSwapInt32(&isWriting, 0, 1) {
		return
	}

	// 获取当前配置的文件名
	cfg := config.GetConfig()
	if cfg == nil {
		atomic.StoreInt32(&isWriting, 0)
		return
	}

	// 查找最新创建的trace数据文件
	// 注意：这是一个临时解决方案，真正的实现应该维护一个全局的文件路径映射
	files, err := filepath.Glob("./data/*_TraceData.csv")
	if err != nil || len(files) == 0 {
		log.WriteLog("Error: Cannot find trace data file")
		atomic.StoreInt32(&isWriting, 0)
		return
	}

	// 获取最新的文件
	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			continue
		}

		if latestFile == "" || fileInfo.ModTime().After(latestTime) {
			latestFile = file
			latestTime = fileInfo.ModTime()
		}
	}

	if latestFile == "" {
		log.WriteLog("Error: Cannot find valid trace data file")
		atomic.StoreInt32(&isWriting, 0)
		return
	}

	// 调用写入函数
	WriteToTraceDataCSV(latestFile)

	// 重置写入状态标志
	atomic.StoreInt32(&isWriting, 0)

	// 手动触发垃圾回收
	runtime.GC()
}

// getTraceData 从车辆获取轨迹数据并格式化为CSV记录
// tag参数用于标记数据来源（如"periodic"、"checkpoint"等）
func getTraceData(vehicle *element.Vehicle, tag string) [][]string {
	trace := vehicle.Trace()
	if len(trace) == 0 {
		return nil // 如果轨迹为空，返回nil
	}

	vehicleID := vehicle.Index()

	// 不再需要这些字段
	// isClosed := vehicle.Flag()
	// pathLength := vehicle.PathLength()
	// origin, destination := int64(-1), int64(-1)
	// if od := vehicle.GetOD(); len(od) >= 2 {
	//    origin = od[0].ID()
	//    destination = od[1].ID()
	// }

	records := make([][]string, 0, len(trace))
	for posID, timeStamp := range trace {
		// 速度和加速度也不再需要
		// speed := vehicle.Velocity()
		// acceleration := vehicle.Acceleration()

		// 精简记录，只保留三个字段
		record := []string{
			strconv.FormatInt(vehicleID, 10), // 车辆 ID
			strconv.Itoa(timeStamp),          // 时间
			strconv.FormatInt(posID, 10),     // 节点 ID
		}
		records = append(records, record)
	}
	return records
}

// InitTraceDataCSV 初始化轨迹数据CSV文件
func InitTraceDataCSV(filename string) {
	// 修改表头，只保留三个字段
	header := []string{
		"VehicleID", "Time", "PosID",
	}
	initializeCSV(filename, header)

	// 更新配置
	UpdateConfig()
}

// WriteToTraceDataCSV 将缓存的轨迹数据写入CSV文件
func WriteToTraceDataCSV(filename string) {
	traceDataMutex.Lock() // 获取写锁，防止并发读写冲突

	// 如果没有数据，直接返回
	if len(traceDataCache) == 0 {
		traceDataMutex.Unlock()
		return
	}

	// 交换缓存，创建一个新的缓存数组
	dataToWrite := traceDataCache

	// 创建新的缓存数组，使用合理的初始容量 - 不再使用maxCacheSize
	// 这里使用当前容量的1/4作为初始容量，避免过大的预分配
	newCapacity := cap(traceDataCache) / 4
	if newCapacity < 1000 {
		newCapacity = 1000 // 最小容量保证
	}
	traceDataCache = make([][][]string, 0, newCapacity)

	// 在释放锁之前完成所有写入操作准备工作
	traceDataMutex.Unlock()

	// 记录写入前的内存状态
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	memBeforeMB := int(memStatsBefore.Alloc / (1024 * 1024))

	// 记录开始写入
	recordsCount := len(dataToWrite)
	log.WriteLog(fmt.Sprintf("Writing %d trace data records to file (Memory: %dMB)", recordsCount, memBeforeMB))
	startTime := time.Now()

	// 处理写入错误
	defer func() {
		if r := recover(); r != nil {
			log.WriteLog(fmt.Sprintf("Error in WriteToTraceDataCSV: %v", r))
		}

		// 读取写入后的内存状态
		var memStatsAfter runtime.MemStats
		runtime.ReadMemStats(&memStatsAfter)
		memAfterMB := int(memStatsAfter.Alloc / (1024 * 1024))

		// 记录写入完成和耗时，以及内存变化
		elapsed := time.Since(startTime)
		log.WriteLog(fmt.Sprintf("Trace data write completed in %v. Memory: %dMB -> %dMB (Δ%dMB)",
			elapsed, memBeforeMB, memAfterMB, memBeforeMB-memAfterMB))
	}()

	// 写入数据到CSV
	for _, records := range dataToWrite {
		appendToCSV(filename, records)
	}

	// 手动释放内存
	dataToWrite = nil

	// 重置记录计数
	atomic.StoreInt32(&recordCount, 0)

	// 手动触发垃圾回收
	runtime.GC()
}

// FormatTraceForNewJourney 将车辆的当前轨迹格式化为记录数据，并标记为行程结束
// 用于处理循环车辆开始新行程前的轨迹数据
// 返回格式化的轨迹数据，可直接保存
func FormatTraceForNewJourney(vehicle *element.Vehicle) [][]string {
	if vehicle == nil {
		return nil
	}

	// 将轨迹格式化为记录数据，使用journey_completed标签
	// 注意：tag参数在新版中不再使用，但为了兼容保留函数调用方式
	traceData := getTraceData(vehicle, "journey_completed")
	return traceData
}

// SaveTraceData 保存轨迹数据到缓存
// 此函数接收的data应该已经是正确格式的轨迹数据（仅包含VehicleID,Time,PosID三个字段）
func SaveTraceData(data [][]string) {
	if len(data) == 0 {
		return
	}

	// 验证数据格式，确保兼容性
	if len(data[0]) > 3 {
		// 如果数据包含超过3个字段，进行转换以适应新格式
		log.WriteLog("Warning: Converting legacy trace data format to new format")
		convertedData := make([][]string, 0, len(data))
		for _, record := range data {
			if len(record) >= 3 {
				// 只保留前三个字段
				convertedData = append(convertedData, record[:3])
			} else if len(record) > 0 {
				// 数据不完整，但尝试保留
				convertedData = append(convertedData, record)
			}
		}
		data = convertedData
	}

	// 使用读写锁的写锁来保护写操作
	traceDataMutex.Lock()
	defer traceDataMutex.Unlock()

	// 将数据包装成轨迹格式并添加到缓存
	wrapper := [][]string(data)
	traceDataCache = append(traceDataCache, wrapper)

	// 增加记录计数
	atomic.AddInt32(&recordCount, 1)
}
