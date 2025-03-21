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

	// 记录模拟参数
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

	// 记录路网参数
	log.WriteLog(fmt.Sprintf("路网类型: %s", cfg.Graph.GraphType))
	if cfg.Graph.GraphType == "cycle" {
		log.WriteLog(fmt.Sprintf("环形路网单元格数: %d", cfg.Graph.CycleGraph.NumCell))
		log.WriteLog(fmt.Sprintf("环形路网红绿灯间隔: %d", cfg.Graph.CycleGraph.LightIndexInterval))
	} else if cfg.Graph.GraphType == "starRing" {
		log.WriteLog(fmt.Sprintf("星形环形路网环形连接单元格数: %d", cfg.Graph.StarRingGraph.RingCellsPerDirection))
		log.WriteLog(fmt.Sprintf("星形环形路网星形连接单元格数: %d", cfg.Graph.StarRingGraph.StarCellsPerDirection))
	}

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
	var g *simple.DirectedGraph
	var nodesMap map[int64]graph.Node
	var lights map[int64]*element.TrafficLightCell

	// 根据配置选择创建的路网类型
	switch cfg.Graph.GraphType {
	case "cycle":
		// 创建环形路网
		g, nodesMap, lights = simulator.CreateCycleGraph(
			cfg.Graph.CycleGraph.NumCell,
			cfg.Graph.CycleGraph.LightIndexInterval,
			cfg.TrafficLight.InitPhaseInterval,
		)
	case "starRing":
		// 创建星形环形混合路网
		g, nodesMap, lights = simulator.CreateStarRingGraph(
			cfg.Graph.StarRingGraph.RingCellsPerDirection,
			cfg.Graph.StarRingGraph.StarCellsPerDirection,
			cfg.TrafficLight.InitPhaseInterval,
		)
	default:
		// 默认创建环形路网
		log.WriteLog(fmt.Sprintf("未知的路网类型: %s，使用默认环形路网", cfg.Graph.GraphType))
		g, nodesMap, lights = simulator.CreateCycleGraph(
			cfg.Graph.CycleGraph.NumCell,
			cfg.Graph.CycleGraph.LightIndexInterval,
			cfg.TrafficLight.InitPhaseInterval,
		)
	}

	numNodes := len(nodesMap)
	log.WriteLog(fmt.Sprintf("路网类型: %s", cfg.Graph.GraphType))
	log.WriteLog(fmt.Sprintf("节点总数: %d", numNodes))
	log.WriteLog(fmt.Sprintf("红绿灯数量: %d", len(lights)))

	// 将map转换为切片，便于后续处理
	nodes := make([]graph.Node, 0, numNodes)
	// 计算所有车道总数
	var allLane float64
	for _, node := range nodesMap {
		nodes = append(nodes, node)
		allLane += node.(element.Cell).Capacity()
	}
	// 计算平均车道数
	avgLane := allLane / float64(numNodes)
	log.WriteLog(fmt.Sprintf("平均车道数: %.2f", avgLane))

	// 检查图是否强连通
	gConnect := utils.IsStronglyConnected(g)
	log.WriteLog(fmt.Sprintf("图连通性: %v", gConnect))

	// 对于starRing图，使用更详细的连通性检查
	if cfg.Graph.GraphType == "starRing" && !gConnect {
		isConnected, problems := simulator.VerifyStarRingGraphConnectivity(g)
		log.WriteLog(fmt.Sprintf("starRing图连通性详细检查: %v", isConnected))
		if !isConnected && len(problems) > 0 {
			log.WriteLog("连通性问题详情:")
			for _, problem := range problems {
				log.WriteLog(fmt.Sprintf("- %s", problem))
			}
		}
	}

	// 创建跟踪节点切片（用于记录轨迹数据）
	var traceNodes []graph.Node
	if cfg.Trace.Enabled {
		// 根据配置决定跟踪节点的采样方式
		if cfg.Trace.TraceRecordInterval > 1 {
			// 每隔固定节点数选择一个作为跟踪节点
			traceNodes = make([]graph.Node, 0, len(nodes)/cfg.Trace.TraceRecordInterval+1)
			for i, node := range nodes {
				if i%cfg.Trace.TraceRecordInterval == 0 {
					traceNodes = append(traceNodes, node)
				}
			}
			log.WriteLog(fmt.Sprintf("轨迹记录已启用，采样间隔: %d, 跟踪节点数: %d", cfg.Trace.TraceRecordInterval, len(traceNodes)))
		} else {
			// 使用所有节点
			traceNodes = nodes
			log.WriteLog("轨迹记录已启用，跟踪所有节点")
		}
	} else {
		traceNodes = nil
		log.WriteLog("轨迹记录已禁用")
	}

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

	// 创建同步写入函数替代异步写入
	writeData := func(writeTrace bool) {
		// 处理数据写入过程中的panic
		defer func() {
			if r := recover(); r != nil {
				log.WriteLog(fmt.Sprintf("Panic occurred during data write: %v", r))
			}
		}()

		// 直接写入数据，不使用工作池和通道
		recorder.WriteToSystemDataCSV(dataFiles["system"])

		// 仅在需要时写入轨迹数据
		if writeTrace && cfg.Trace.Enabled {
			recorder.WriteToTraceDataCSV(dataFiles["trace"])
		}

		recorder.WriteToVehicleDataCSV(dataFiles["vehicle"])

		// 手动触发垃圾回收以减少内存占用
		runtime.GC()
	}

	// 更新trace recorder配置
	recorder.UpdateConfig()

	// 初始化计数器，用于跟踪上次trace写入时间
	lastTraceWriteStep := 0

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

		// 更新系统状态
		sysState.Update(nodes, numNodes, avgLane)
		sysState.RecordData(timeStep)

		// 按间隔输出日志
		if timeOfDay%cfg.Logging.IntervalWriteToLog == 0 {
			sysState.LogStatus(currentDay, timeOfDay)
		}

		// 检查是否应该写入trace数据（使用独立的写入间隔）
		needWriteTrace := false
		if cfg.Trace.Enabled && cfg.Trace.WriteInterval > 0 {
			if timeStep-lastTraceWriteStep >= cfg.Trace.WriteInterval {
				needWriteTrace = true
				lastTraceWriteStep = timeStep
			}
		}

		// 按间隔写入系统和车辆数据
		if timeOfDay%cfg.Logging.IntervalWriteOtherData == 0 {
			// 只在需要时写入trace数据
			writeData(needWriteTrace)
		} else if needWriteTrace {
			// 如果只需要写入trace数据
			writeData(true)
		}
	}

	// 在模拟结束时执行一次最终数据写入，确保所有数据写入
	writeData(true)

	// 不再需要等待信号量，直接进入finishSimulation
}

// 完成模拟，写入最后的数据
func finishSimulation(pool *WorkerPool, dataFiles map[string]string) {
	log.WriteLog("Writing final data...")

	// 直接在主线程中执行最后的写入操作
	// 处理数据写入过程中的panic
	defer func() {
		if r := recover(); r != nil {
			log.WriteLog(fmt.Sprintf("Panic occurred during final data write: %v", r))
		}
	}()

	// 确保最后trace recorder配置被更新
	recorder.UpdateConfig()

	// 同步执行数据写入
	startTime := time.Now()

	recorder.WriteToSystemDataCSV(dataFiles["system"])
	recorder.WriteToTraceDataCSV(dataFiles["trace"])
	recorder.WriteToVehicleDataCSV(dataFiles["vehicle"])

	elapsedTime := time.Since(startTime)
	log.WriteLog(fmt.Sprintf("Final data write completed in %v", elapsedTime))

	// 手动触发垃圾回收以释放内存
	runtime.GC()
}
