package main

import (
	"context"
	"fmt"
	"graphCA/config"
	"graphCA/element"
	"graphCA/log"
	"graphCA/recorder"
	"graphCA/simulator"
	"graphCA/utils"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// WorkerPool 表示一个工作池
type WorkerPool struct {
	jobs    chan func()
	wg      sync.WaitGroup
	workers int
	closed  atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWorkerPool 创建一个新的工作池
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		jobs:    make(chan func(), workers*2), // 缓冲区大小为工作者数量的2倍
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
	pool.Start()
	return pool
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					job()
				}
			}
		}()
	}
}

// Submit 提交一个任务到工作池
// 如果工作池已关闭，返回false，否则返回true
func (p *WorkerPool) Submit(job func()) bool {
	if p.closed.Load() {
		return false
	}

	select {
	case p.jobs <- job:
		return true
	case <-p.ctx.Done():
		return false
	}
}

// Stop 停止工作池
// 安全地停止所有工作协程并等待它们完成
func (p *WorkerPool) Stop() {
	// 如果已经关闭，直接返回
	if p.closed.Swap(true) {
		return
	}

	// 取消上下文，通知所有工作协程退出
	p.cancel()

	// 关闭通道前确保所有工作协程已退出循环
	close(p.jobs)

	// 等待所有工作协程完成
	p.wg.Wait()
}

// 缓存常用的系统状态
type SystemState struct {
	numVehicleGenerated int64
	numVehiclesActive   int64
	numVehiclesWaiting  int64
	numVehicleCompleted int64
	vehiclesOnRoad      map[*element.Vehicle]struct{}
	averageSpeed        float64
	density             float64
	mu                  sync.RWMutex // 保护并发访问
}

// 更新系统状态
func (s *SystemState) Update(nodes []graph.Node, numNodes int, avgLane float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.numVehicleGenerated, s.numVehiclesActive, s.numVehiclesWaiting, s.numVehicleCompleted = simulator.GetVehiclesNum()
	s.vehiclesOnRoad = simulator.GetVehiclesOnRoad(nodes)
	s.averageSpeed, s.density = simulator.GetAverageSpeed_Density(s.vehiclesOnRoad, numNodes, avgLane)
}

// 记录系统状态数据
func (s *SystemState) RecordData(timeStep int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recorder.RecordSystemData(timeStep, s.numVehicleGenerated, s.numVehiclesActive,
		s.numVehiclesWaiting, s.numVehicleCompleted, s.averageSpeed, s.density)
}

// 输出系统状态日志
func (s *SystemState) LogStatus(currentDay int, timeOfDay int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.WriteLog(fmt.Sprintf("Day: %d, TimeOfDay: %v, AvgSpeed: %.2f, Density: %.2f, Generated: %d, Active: %d, OnRoad: %d, Waiting: %d, Completed: %d",
		currentDay, log.ConvertTimeStepToTime(timeOfDay), s.averageSpeed, s.density,
		s.numVehicleGenerated, s.numVehiclesActive, len(s.vehiclesOnRoad),
		s.numVehiclesWaiting, s.numVehicleCompleted))
}

// VehicleManager is an interface for getting vehicle numbers
type VehicleManager interface {
	GetVehiclesNum() (int64, int64, int64, int64)
	// Add any other necessary methods here
}

func main() {
	// 加载配置文件
	if err := config.LoadConfig("config/config.json"); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	cfg := config.GetConfig()

	// 生成唯一的初始化时间标识
	initTime := time.Now().Format("2006010215040506")

	// 初始化资源
	pool, _, dataFiles := initializeResources(cfg, initTime)
	defer func() {
		log.WriteLog("正在停止工作池...")
		pool.Stop()
		log.WriteLog("工作池已停止")
		log.CloseLog()
	}()

	// 初始化模拟环境
	g, nodes, lights, traceNodes, avgLane := initializeSimulationEnvironment(cfg)
	numNodes := len(nodes)

	// 初始化系统状态
	sysState := &SystemState{
		vehiclesOnRoad: make(map[*element.Vehicle]struct{}),
	}
	var demand []float64

	// 初始化车辆
	simulator.InitFixedVehicle(cfg.Vehicle.NumClosedVehicle, g, nodes, traceNodes)

	// 创建数据写入通道和异步写入函数
	dataWriteChan := createDataWriteChannel(dataFiles)

	// 开始模拟
	log.WriteLog("----------------------------------Simulation Start----------------------------------")
	runSimulation(cfg, g, nodes, lights, traceNodes, numNodes, avgLane, sysState, &demand, pool, dataWriteChan, dataFiles)

	// 完成模拟，写入最后的数据
	finishSimulation(pool, dataFiles)

	log.WriteLog("---------------------------------- Completed ----------------------------------")
}

// 初始化系统资源
func initializeResources(cfg *config.Config, initTime string) (*WorkerPool, string, map[string]string) {
	// 初始化工作池
	numWorkers := runtime.GOMAXPROCS(0)
	pool := NewWorkerPool(numWorkers)

	// 日志初始化
	logFile := fmt.Sprintf("./log/%s_%d.log", initTime, cfg.Vehicle.NumClosedVehicle)
	log.InitLog(logFile)
	log.LogEnvironment()
	log.LogSimParameters(
		cfg.Simulation.OneDayTimeSteps,
		cfg.Demand.Multiplier,
		cfg.Demand.FixedNum,
		cfg.Demand.RandomDisRange,
		cfg.Vehicle.NumClosedVehicle,
		cfg.Simulation.SimDay,
		cfg.TrafficLight.InitPhaseInterval,
		cfg.TrafficLight.Changes[0].Day,
		cfg.TrafficLight.Changes[0].Multiplier,
		cfg.TrafficLight.Changes[1].Day,
		cfg.TrafficLight.Changes[1].Multiplier,
	)
	log.WriteLog(fmt.Sprintf("Concurrent Volume in Vehicle Process: %d", numWorkers))

	// 数据CSV初始化
	systemDataFile := fmt.Sprintf("./data/%s_%d_SystemData.csv", initTime, cfg.Vehicle.NumClosedVehicle)
	vehicleDataFile := fmt.Sprintf("./data/%s_%d_VehicleData.csv", initTime, cfg.Vehicle.NumClosedVehicle)
	traceDataFile := fmt.Sprintf("./data/%s_%d_TraceData.csv", initTime, cfg.Vehicle.NumClosedVehicle)

	recorder.InitSystemDataCSV(systemDataFile)
	recorder.InitVehicleDataCSV(vehicleDataFile)
	recorder.InitTraceDataCSV(traceDataFile)

	dataFiles := map[string]string{
		"system":  systemDataFile,
		"vehicle": vehicleDataFile,
		"trace":   traceDataFile,
	}

	return pool, logFile, dataFiles
}

// 初始化模拟环境
func initializeSimulationEnvironment(cfg *config.Config) (*simple.DirectedGraph, []graph.Node, map[int64]*element.TrafficLightCell, []graph.Node, float64) {
	// 仿真图初始化
	g, nodesMap, lights := simulator.CreateCycleGraph(
		cfg.Simulation.NumCell,
		cfg.Simulation.LightIndexInterval,
		cfg.TrafficLight.InitPhaseInterval,
	)
	numNodes := len(nodesMap)
	log.WriteLog(fmt.Sprintf("Number of Nodes: %d", numNodes))
	log.WriteLog(fmt.Sprintf("Number of TrafficLight Group: %d", len(lights)))

	// 预分配节点切片以减少内存分配
	nodes := make([]graph.Node, len(nodesMap))
	var allLane float64
	for i, node := range nodesMap {
		nodes[i] = node
		allLane += node.(element.Cell).Capacity()
	}
	avgLane := allLane / float64(numNodes)
	log.WriteLog(fmt.Sprintf("Average lanes: %.2f", avgLane))

	gConnect := utils.IsStronglyConnected(g)
	log.WriteLog(fmt.Sprintf("Graph Connected: %v", gConnect))

	// 跟踪节点初始化 - 选择更多的节点进行轨迹记录
	// 每隔固定节点数选择一个作为跟踪节点
	trackNodeInterval := 20 // 固定值，不再从配置中读取
	log.WriteLog(fmt.Sprintf("Track node interval: %d", trackNodeInterval))

	traceNodes := make([]graph.Node, 0, len(nodes)/trackNodeInterval+1)

	for i, node := range nodes {
		if i%trackNodeInterval == 0 {
			traceNodes = append(traceNodes, node)
		}
	}

	log.WriteLog(fmt.Sprintf("Total trace nodes: %d", len(traceNodes)))

	return g, nodes, lights, traceNodes, avgLane
}

// 创建数据写入通道
func createDataWriteChannel(dataFiles map[string]string) chan struct{} {
	// 使用缓冲大小为1的通道以实现信号量功能
	return make(chan struct{}, 1)
}

// 运行模拟
func runSimulation(cfg *config.Config, g *simple.DirectedGraph, nodes []graph.Node, lights map[int64]*element.TrafficLightCell,
	traceNodes []graph.Node, numNodes int, avgLane float64, sysState *SystemState, demand *[]float64,
	pool *WorkerPool, dataWriteChan chan struct{}, dataFiles map[string]string) {

	simDaySteps := cfg.Simulation.SimDay * cfg.Simulation.OneDayTimeSteps

	// 创建停止周期性轨迹记录的通道
	traceRecordingStopChan := make(chan struct{})

	// 根据配置决定是否启动轨迹记录
	if cfg.Trace.Enabled {
		// 启动周期性轨迹记录（后台运行）
		// 设置固定的扫描间隔
		traceRecordInterval := 1200 // 固定值，不再从配置中读取
		log.WriteLog(fmt.Sprintf("Trace recording enabled. Record interval: %d", traceRecordInterval))

		go recorder.PeriodicTraceRecording(traceRecordInterval, nodes, traceRecordingStopChan)
	} else {
		log.WriteLog("Trace recording disabled in configuration")
	}

	// 确保在函数返回时停止轨迹记录
	defer func() {
		close(traceRecordingStopChan)
	}()

	// 设置快照间隔的固定值
	snapshotInterval := 57600 // 固定值，不再从配置中读取
	log.WriteLog(fmt.Sprintf("Snapshot interval: %d", snapshotInterval))

	// 创建异步写入函数
	writeDataAsync := func() {
		select {
		case dataWriteChan <- struct{}{}:
			if pool.Submit(func() {
				defer func() {
					select {
					case <-dataWriteChan:
					default:
						// 防止在通道已关闭的情况下出现panic
					}
				}()

				// 处理数据写入过程中的panic
				defer func() {
					if r := recover(); r != nil {
						log.WriteLog(fmt.Sprintf("Panic occurred during data write: %v", r))
					}
				}()

				recorder.WriteToSystemDataCSV(dataFiles["system"])
				// 仅在启用轨迹记录时写入轨迹数据
				if cfg.Trace.Enabled {
					recorder.WriteToTraceDataCSV(dataFiles["trace"])
				}
				recorder.WriteToVehicleDataCSV(dataFiles["vehicle"])
			}) {
				// 提交成功
			} else {
				// 工作池已关闭，手动释放信号量
				<-dataWriteChan
				log.WriteLog("Warning: Worker pool closed, data write task not submitted")
			}
		default:
			// 如果上一次写入还未完成，跳过本次写入
			log.WriteLog("Warning: Previous data write not completed, skipping current write")
		}
	}

	// 模拟主循环
	for timeStep := 0; timeStep < simDaySteps; timeStep++ {
		timeOfDay := timeStep % cfg.Simulation.OneDayTimeSteps
		currentDay := timeStep/cfg.Simulation.OneDayTimeSteps + 1

		// 每天开始时更新需求分布
		if timeOfDay == 0 {
			*demand = simulator.AdjustDemand(
				cfg.Demand.Multiplier,
				cfg.Demand.FixedNum,
				cfg.Demand.DayRandomDisRange,
			)
		}

		// 红绿灯周期改变检查
		for _, change := range cfg.TrafficLight.Changes {
			if currentDay == change.Day && timeOfDay == 0 {
				for _, light := range lights {
					light.ChangeInterval(change.Multiplier)
				}
				log.WriteLog(fmt.Sprintf("TrafficLight Interval Changed: Multiplier - %.2f", change.Multiplier))
			}
		}

		// 生成和处理车辆
		generateNum := simulator.GetGenerateVehicleCount(timeOfDay, *demand, cfg.Demand.RandomDisRange)
		simulator.GenerateScheduleVehicle(timeStep, generateNum, g, nodes, traceNodes)

		// 红绿灯循环
		for _, light := range lights {
			light.Cycle()
		}

		// 处理车辆移动
		simulator.VehicleProcess(runtime.GOMAXPROCS(0), timeStep, g, traceNodes)

		// 仅当启用轨迹记录时，才执行快照记录
		if cfg.Trace.Enabled && timeStep > 0 && timeStep%snapshotInterval == 0 {
			recorder.ScanNetworkForVehicles(timeStep, nodes, "snapshot")
		}

		// 更新系统状态
		sysState.Update(nodes, numNodes, avgLane)
		sysState.RecordData(timeStep)

		// 按间隔输出日志
		if timeOfDay%cfg.Logging.IntervalWriteToLog == 0 {
			sysState.LogStatus(currentDay, timeOfDay)
		}

		// 按间隔异步写入数据
		if timeOfDay%cfg.Logging.IntervalWriteOtherData == 0 {
			writeDataAsync()
		}
	}

	// 在模拟结束时执行一次最终数据写入
	writeDataAsync()

	// 确保最后的数据写入任务完成
	// 等待信号量释放，表示任务已完成
	timeout := time.After(5 * time.Second)
	select {
	case <-timeout:
		log.WriteLog("Warning: Timeout waiting for final data write to complete")
	case <-dataWriteChan:
		// 信号量被释放，表示写入已完成
		// 立即放回信号量以避免死锁
		select {
		case dataWriteChan <- struct{}{}:
		default:
		}
	}
}

// 完成模拟，写入最后的数据
func finishSimulation(pool *WorkerPool, dataFiles map[string]string) {
	log.WriteLog("Writing final data...")

	// 创建等待组以确保最后的数据写入完成
	wg := sync.WaitGroup{}
	wg.Add(1)

	// 提交最后的写入任务
	if !pool.Submit(func() {
		defer wg.Done()

		// 处理数据写入过程中的panic
		defer func() {
			if r := recover(); r != nil {
				log.WriteLog(fmt.Sprintf("Panic occurred during final data write: %v", r))
			}
		}()

		recorder.WriteToSystemDataCSV(dataFiles["system"])
		recorder.WriteToTraceDataCSV(dataFiles["trace"])
		recorder.WriteToVehicleDataCSV(dataFiles["vehicle"])
	}) {
		// 如果提交失败，手动将等待组计数减一
		wg.Done()
		log.WriteLog("Warning: Final data write task not submitted, worker pool may be closed")
	}

	// 等待最后的数据写入完成
	wg.Wait()
	log.WriteLog("Final data write completed")
}
