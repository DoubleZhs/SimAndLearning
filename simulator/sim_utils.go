package simulator

import (
	"fmt"
	"runtime"
	"simAndLearning/log"
	"simAndLearning/recorder"
	"time"
)

// WriteData 同步写入系统和车辆数据
// 处理数据写入过程中可能出现的panic
func WriteData(dataFiles map[string]string) {
	// 处理数据写入过程中的panic
	defer func() {
		if r := recover(); r != nil {
			log.WriteLog(fmt.Sprintf("Panic occurred during data write: %v", r))
		}
	}()

	// 直接写入数据
	recorder.WriteToSystemDataCSV(dataFiles["system"])
	recorder.WriteToVehicleDataCSV(dataFiles["vehicle"])
	// 写入轨迹数据
	if traceFile, ok := dataFiles["trace"]; ok {
		recorder.WriteToTraceDataCSV(traceFile)
	}

	// 手动触发垃圾回收以减少内存占用
	runtime.GC()
}

// FinishSimulation 完成模拟，写入最后的数据
// 记录写入操作的时间消耗
func FinishSimulation(dataFiles map[string]string) {
	log.WriteLog("Writing final data...")

	// 直接在主线程中执行最后的写入操作
	// 处理数据写入过程中的panic
	defer func() {
		if r := recover(); r != nil {
			log.WriteLog(fmt.Sprintf("Panic occurred during final data write: %v", r))
		}
	}()

	// 同步执行数据写入
	startTime := time.Now()

	recorder.WriteToSystemDataCSV(dataFiles["system"])
	recorder.WriteToVehicleDataCSV(dataFiles["vehicle"])
	// 写入轨迹数据
	if traceFile, ok := dataFiles["trace"]; ok {
		recorder.WriteToTraceDataCSV(traceFile)
	}

	elapsedTime := time.Since(startTime)
	log.WriteLog(fmt.Sprintf("Final data write completed in %v", elapsedTime))

	// 手动触发垃圾回收以释放内存
	runtime.GC()
}
