package main

import (
	"fmt"
	"graphCA/config"
	"graphCA/element"
	"graphCA/log"
	"graphCA/recorder"
	"graphCA/simulator"
	"graphCA/utils"
	"runtime"
	"sync"
	"time"

	"gonum.org/v1/gonum/graph"
)

// WorkerPool 表示一个工作池
type WorkerPool struct {
	jobs    chan func()
	wg      sync.WaitGroup
	workers int
	closed  bool
	mu      sync.Mutex
}

// NewWorkerPool 创建一个新的工作池
func NewWorkerPool(workers int) *WorkerPool {
	pool := &WorkerPool{
		jobs:    make(chan func(), workers*2), // 缓冲区大小为工作者数量的2倍
		workers: workers,
		closed:  false,
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
			for job := range p.jobs {
				job()
			}
		}()
	}
}

// Submit 提交一个任务到工作池
func (p *WorkerPool) Submit(job func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed {
		p.jobs <- job
	}
}

// Stop 停止工作池
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.jobs)
		p.mu.Unlock()
		p.wg.Wait()
	} else {
		p.mu.Unlock()
	}
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
}

// 更新系统状态
func (s *SystemState) Update(nodes []graph.Node, numNodes int, avgLane float64) {
	s.numVehicleGenerated, s.numVehiclesActive, s.numVehiclesWaiting, s.numVehicleCompleted = simulator.GetVehiclesNum()
	s.vehiclesOnRoad = simulator.GetVehiclesOnRoad(nodes)
	s.averageSpeed, s.density = simulator.GetAverageSpeed_Density(s.vehiclesOnRoad, numNodes, avgLane)
}

// 记录系统状态数据
func (s *SystemState) RecordData(timeStep int) {
	recorder.RecordSystemData(timeStep, s.numVehicleGenerated, s.numVehiclesActive,
		s.numVehiclesWaiting, s.numVehicleCompleted, s.averageSpeed, s.density)
}

// 输出系统状态日志
func (s *SystemState) LogStatus(currentDay int, timeOfDay int) {
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

	initTime := time.Now().Format("2006010215040506")

	// 初始化工作池
	numWorkers := runtime.GOMAXPROCS(0)
	pool := NewWorkerPool(numWorkers)
	// 只保留一个pool.Stop()调用，因为defer会在函数结束前执行
	defer pool.Stop()

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
	defer log.CloseLog()

	// 数据CSV初始化
	systemDataFile := fmt.Sprintf("./data/%s_%d_SystemData.csv", initTime, cfg.Vehicle.NumClosedVehicle)
	vehicleDataFile := fmt.Sprintf("./data/%s_%d_VehicleData.csv", initTime, cfg.Vehicle.NumClosedVehicle)
	traceDataFile := fmt.Sprintf("./data/%s_%d_TraceData.csv", initTime, cfg.Vehicle.NumClosedVehicle)
	recorder.InitSystemDataCSV(systemDataFile)
	recorder.InitVehicleDataCSV(vehicleDataFile)
	recorder.InitTraceDataCSV(traceDataFile)

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

	// 预分配追踪节点切片
	traceNodes := make([]graph.Node, 0, len(nodes)/80+1)
	for i, node := range nodes {
		if i%80 == 0 {
			traceNodes = append(traceNodes, node)
		}
	}

	// 系统初始化
	var demand []float64
	simulator.InitFixedVehicle(cfg.Vehicle.NumClosedVehicle, g, nodes, traceNodes)

	// 创建系统状态缓存
	var sysState SystemState

	// 创建数据写入通道并添加缓冲区
	dataWriteChan := make(chan struct{}, 1)
	writeDataAsync := func() {
		select {
		case dataWriteChan <- struct{}{}:
			pool.Submit(func() {
				recorder.WriteToSystemDataCSV(systemDataFile)
				recorder.WriteToTraceDataCSV(traceDataFile)
				recorder.WriteToVehicleDataCSV(vehicleDataFile)
				<-dataWriteChan
			})
		default:
			// 如果上一次写入还未完成，跳过本次写入
			log.WriteLog("Warning: Previous data write not completed, skipping current write")
		}
	}

	// 仿真主进程
	log.WriteLog("----------------------------------Simulation Start----------------------------------")
	simDaySteps := cfg.Simulation.SimDay * cfg.Simulation.OneDayTimeSteps

	// 模拟主循环
	for timeStep := 0; timeStep < simDaySteps; timeStep++ {
		timeOfDay := timeStep % cfg.Simulation.OneDayTimeSteps
		currentDay := timeStep/cfg.Simulation.OneDayTimeSteps + 1

		// 每天开始时更新需求分布
		if timeOfDay == 0 {
			demand = simulator.AdjustDemand(
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
		generateNum := simulator.GetGenerateVehicleCount(timeOfDay, demand, cfg.Demand.RandomDisRange)
		simulator.GenerateScheduleVehicle(timeStep, generateNum, g, nodes, traceNodes)

		// 红绿灯循环
		for _, light := range lights {
			light.Cycle()
		}

		// 处理车辆移动
		simulator.VehicleProcess(numWorkers, timeStep, g, traceNodes)

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

	// 确保所有数据都被写入
	pool.Submit(func() {
		recorder.WriteToSystemDataCSV(systemDataFile)
		recorder.WriteToTraceDataCSV(traceDataFile)
		recorder.WriteToVehicleDataCSV(vehicleDataFile)
	})

	log.WriteLog("---------------------------------- Completed ----------------------------------")
}
