package element

import (
	"errors"
	"graphCA/config"
	"math/rand/v2"
	"sync"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// Vehicle 表示一个车辆
type Vehicle struct {
	index         int64                 // 车辆唯一标识
	velocity      int                   // 当前速度
	acceleration  int                   // 加速度
	occupy        float64               // 占用空间
	slowingProb   float64               // 随机减速概率
	tag           float64               // 车辆标签，用于随机化处理
	flag          bool                  // 标记车辆是否是固定车辆
	state         int                   // 车辆状态 (1=设置起终点, 2=设置路径, 3=进入缓冲区, 4=系统中, 5=完成)
	graph         *simple.DirectedGraph // 路网图
	pos           graph.Node            // 当前位置
	origin        graph.Node            // 起点
	destination   graph.Node            // 终点
	simplePath    []graph.Node          // 简化路径
	residualPath  []graph.Node          // 剩余路径
	pathlength    int                   // 路径长度
	inTime        int                   // 进入系统时间
	outTime       int                   // 离开系统时间
	trace         map[int64]int         // 轨迹记录 (节点ID -> 时间)，记录车辆经过的位置和时间
	traceInterval int                   // 轨迹记录间隔（时间步），0表示使用默认设置
	lastTraceTime int                   // 上次记录轨迹的时间
	activiate     bool                  // 是否激活
	mu            sync.RWMutex          // 用于保护并发访问
}

// NewVehicle 创建一个新车辆
func NewVehicle(index int64, velocity, acceleration int, occupy, slowingProb float64, flag bool) *Vehicle {
	if velocity < 0 {
		panic("velocity must be non-negative")
	}
	if acceleration < 0 {
		panic("acceleration must be non-negative")
	}
	if occupy <= 0 {
		panic("occupy must be positive")
	}
	if slowingProb < 0 || slowingProb > 1 {
		panic("slowing probability must be between 0 and 1")
	}

	return &Vehicle{
		index:         index,
		velocity:      velocity,
		acceleration:  acceleration,
		occupy:        occupy,
		slowingProb:   slowingProb,
		tag:           rand.Float64(),
		flag:          flag,
		trace:         make(map[int64]int),
		lastTraceTime: 0,
	}
}

// Index 返回车辆ID
func (v *Vehicle) Index() int64 {
	return v.index
}

// Velocity 返回车辆当前速度
func (v *Vehicle) Velocity() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.velocity
}

// Acceleration 返回车辆加速度
func (v *Vehicle) Acceleration() int {
	return v.acceleration
}

// SlowingProb 返回车辆随机减速概率
func (v *Vehicle) SlowingProb() float64 {
	return v.slowingProb
}

// State 返回车辆当前状态
func (v *Vehicle) State() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.state
}

// Flag 返回车辆是否是固定车辆
func (v *Vehicle) Flag() bool {
	return v.flag
}

// Origin 返回车辆起点
func (v *Vehicle) Origin() graph.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.origin
}

// Destination 返回车辆终点
func (v *Vehicle) Destination() graph.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.destination
}

// Path 返回车辆路径
func (v *Vehicle) Path() []graph.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// 返回副本以避免外部修改
	result := make([]graph.Node, len(v.simplePath))
	copy(result, v.simplePath)
	return result
}

// PathLength 返回车辆路径长度
func (v *Vehicle) PathLength() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.pathlength
}

// Trace 返回车辆轨迹
func (v *Vehicle) Trace() map[int64]int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// 返回副本以避免外部修改
	result := make(map[int64]int, len(v.trace))
	for k, v := range v.trace {
		result[k] = v
	}
	return result
}

// Report 返回车辆基本信息
func (v *Vehicle) Report() (int64, int, float64, int, int, float64, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.index, v.acceleration, v.slowingProb, v.inTime, v.outTime, v.tag, v.flag
}

// SetOD 设置车辆的起点和终点
func (v *Vehicle) SetOD(g *simple.DirectedGraph, origin, destination graph.Node) (bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if origin.ID() == destination.ID() {
		return false, errors.New("origin and destination are the same")
	}
	v.graph = g
	v.origin = origin
	v.destination = destination
	v.state = 1
	return true, nil
}

// SetPath 设置车辆路径
func (v *Vehicle) SetPath(path []graph.Node) (bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.state != 1 {
		return false, errors.New("set origin and destination first")
	}

	if len(path) == 0 {
		return false, errors.New("path cannot be empty")
	}

	if path[0] != v.origin {
		return false, errors.New("path does not start from origin")
	}

	if path[len(path)-1] != v.destination {
		return false, errors.New("path does not end at destination")
	}

	v.simplePath = path
	v.residualPath = make([]graph.Node, 0, len(path)*2) // 预估扩展后的路径长度

	// 展开路径中的链路
	for _, node := range path {
		switch assertNode := node.(type) {
		case Cell:
			v.residualPath = append(v.residualPath, assertNode)
		case *Link:
			v.residualPath = append(v.residualPath, assertNode.Flat()...)
		default:
			return false, errors.New("node is not a cell or link")
		}
	}

	v.pathlength = len(v.residualPath)
	v.state = 2
	return true, nil
}

// BufferIn 将车辆添加到起点的缓冲区
func (v *Vehicle) BufferIn(inTime int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.state != 2 {
		panic("set path first")
	}

	cell, ok := (v.origin).(Cell)
	if !ok {
		panic("origin is not a cell")
	}

	cell.BufferLoad(v)
	v.inTime = inTime
	v.state = 3
}

// UpdateActiveState 更新车辆激活状态
func (v *Vehicle) UpdateActiveState() bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	originCell, ok := (v.origin).(Cell)
	if !ok {
		panic("origin is not a cell")
	}

	totalOccupation := originCell.Occupation()
	for _, vehicle := range originCell.ListBuffer() {
		totalOccupation += vehicle.occupy
		if totalOccupation > originCell.Capacity() {
			v.activiate = false
			return false
		}
		if vehicle == v {
			v.activiate = true
			return true
		}
	}

	v.activiate = false
	return false
}

// SystemIn 将车辆从缓冲区移动到系统中
func (v *Vehicle) SystemIn() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.state != 3 {
		panic("buffer in first")
	}

	if !v.activiate {
		panic("vehicle not activated")
	}

	cell, ok := (v.origin).(Cell)
	if !ok {
		panic("origin is not a cell")
	}

	cell.BufferUnload(v)
	cell.Load(v)
	v.pos = cell
	v.residualPath = v.residualPath[1:]
	v.state = 4
}

// SystemOut 将车辆从系统中移除
// 外部调用此方法，不再由Move方法内部调用
func (v *Vehicle) SystemOut(time int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 如果车辆已经被Move方法处理为状态5，则只需卸载车辆
	if v.state == 5 {
		cell, ok := (v.pos).(Cell)
		if !ok {
			panic("pos is not a cell")
		}

		cell.Unload(v)
		// 不需要再次设置时间和状态，因为Move方法已经设置了
		return
	}

	// 如果车辆还没有被Move方法处理为状态5，执行完整的SystemOut逻辑
	if v.state != 4 {
		panic("system in first")
	}

	cell, ok := (v.pos).(Cell)
	if !ok {
		panic("pos is not a cell")
	}

	cell.Unload(v)
	v.outTime = time
	v.state = 5
}

// Move 移动车辆
// 返回true表示车辆已到达终点
func (v *Vehicle) Move(time int) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 如果是车辆的第一次移动（刚进入系统），记录起点
	if v.state >= 3 && len(v.trace) == 0 && v.origin != nil {
		// 记录起点 - 对于state=3(缓冲区)和state=4(刚进入路网)都记录起点
		v.trace[v.origin.ID()] = time
		v.lastTraceTime = time
	}

	// 如果车辆不在路网中（state!=4），不进行移动
	if v.state != 4 {
		return false
	}

	// 纳格尔(Nagel-Schreckenberg)模型的四个步骤
	for {
		v.accelerate()
		v.decelerate()
		v.randomSlowing()

		if v.velocity == 0 {
			// 即使速度为0，也应考虑记录当前位置
			shouldRecord := (time - v.lastTraceTime) >= v.traceInterval
			if shouldRecord && v.pos != nil {
				v.trace[v.pos.ID()] = time
				v.lastTraceTime = time
			}
			return false
		}

		// 确保索引有效
		if v.velocity > len(v.residualPath) {
			v.velocity = len(v.residualPath)
		}

		targetIndex := v.velocity - 1
		target := v.residualPath[targetIndex]
		targetCell, ok := target.(Cell)

		if !ok {
			panic("target is not a cell")
		}

		if !targetCell.Loadable(v) {
			continue
		}

		// 执行移动
		currentCell, ok := (v.pos).(Cell)
		if !ok {
			panic("current position is not a cell")
		}

		currentCell.Unload(v)
		targetCell.Load(v)
		v.pos = targetCell

		// 记录轨迹 - 判断是否需要记录当前位置
		// 使用未锁定的方法判断，因为外层已经持有锁
		shouldRecord := false

		// 如果是终点，必须记录
		if len(v.residualPath) == v.velocity && v.pos.ID() == v.destination.ID() {
			shouldRecord = true
		} else {
			// 否则根据时间间隔决定是否记录
			interval := v.traceInterval
			if interval <= 0 {
				// 根据配置更新轨迹记录间隔
				cfg := config.GetConfig()
				if cfg != nil && cfg.Trace.Enabled {
					// 使用配置中的轨迹记录间隔
					interval = cfg.Trace.TraceRecordInterval
					v.traceInterval = interval // 更新车辆的轨迹记录间隔
				} else {
					// 默认值
					interval = 10
					v.traceInterval = interval
				}
			}

			shouldRecord = time-v.lastTraceTime >= interval
		}

		// 如果应该记录，则记录当前位置
		if shouldRecord {
			v.trace[v.pos.ID()] = time
			v.lastTraceTime = time
		}

		// 更新路径
		v.residualPath = v.residualPath[v.velocity:]

		// 检查是否到达终点
		if len(v.residualPath) == 0 {
			// 修改：在Move方法中不直接调用SystemOut，而是设置状态
			// 这样可以避免嵌套锁问题
			v.outTime = time
			v.state = 5

			// 确保记录终点
			v.trace[v.pos.ID()] = time
			v.lastTraceTime = time

			// 注意：不再调用v.SystemOut(time)
			return true
		}

		return false
	}
}

// 以下是内部辅助方法

// accelerate 车辆加速
func (v *Vehicle) accelerate() {
	cell, ok := v.pos.(Cell)
	if !ok {
		panic("pos is not a cell")
	}
	v.velocity = min(v.velocity+v.acceleration, cell.MaxSpeed())
}

// decelerate 车辆减速
func (v *Vehicle) decelerate() {
	gap := v.calculateGap()
	v.velocity = min(v.velocity, gap)
}

// calculateGap 计算前方安全距离
func (v *Vehicle) calculateGap() int {
	gap := 0
	maxCheck := min(v.velocity, len(v.residualPath))

	for i := 0; i < maxCheck; i++ {
		node := v.residualPath[i]
		cell, ok := node.(Cell)
		if !ok {
			panic("node is not a cell")
		}

		if !cell.Loadable(v) {
			break
		}

		// 检查是否是交叉路口(入度>1)
		toNodes := v.graph.To(node.ID())
		inDegree := 0
		for toNodes.Next() {
			inDegree++
		}

		// 交叉路口有通过概率
		if inDegree > 1 {
			passProbability := 0.8
			if rand.Float64() > passProbability {
				return gap
			}
		}

		gap++
	}

	return gap
}

// randomSlowing 随机减速
func (v *Vehicle) randomSlowing() {
	if rand.Float64() < v.slowingProb {
		v.velocity = max(v.velocity-1, 0)
	}
}

// func fillPath(g *simple.DirectedGraph, nodes []graph.Node, cache *linkNodesCache) []graph.Node {
// 	results := make([][]graph.Node, len(nodes)-1)
// 	var wg sync.WaitGroup

// 	for i := 0; i < len(nodes)-1; i++ {
// 		wg.Add(1)
// 		go func(i int) {
// 			defer wg.Done()
// 			origin := nodes[i]
// 			destination := nodes[i+1]

// 			// Check cache first
// 			if cachedPath, found := cache.Get(origin, destination); found {
// 				if i > 0 {
// 					cachedPath = cachedPath[1:] // Remove the first node to avoid duplication
// 				}
// 				results[i] = cachedPath
// 				return
// 			}

// 			// Compute shortest path if not found in cache
// 			shortestPath, _, err := utils.ShortestPath(g, origin, destination)
// 			if err != nil {
// 				panic(err)
// 			}

// 			// Store the result in cache
// 			cache.Set(origin, destination, shortestPath)

// 			if i > 0 {
// 				shortestPath = shortestPath[1:] // Remove the first node to avoid duplication
// 			}

// 			results[i] = shortestPath
// 		}(i)
// 	}

// 	wg.Wait()

// 	var fullPath []graph.Node
// 	for _, segment := range results {
// 		fullPath = append(fullPath, segment...)
// 	}

// 	return fullPath
// }

// GetOD 返回车辆的起点和终点
func (v *Vehicle) GetOD() []graph.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.origin == nil || v.destination == nil {
		return nil
	}

	return []graph.Node{v.origin, v.destination}
}

// GetPath 返回车辆的完整路径
func (v *Vehicle) GetPath() []graph.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.simplePath == nil {
		return nil
	}

	// 返回副本以避免外部修改
	result := make([]graph.Node, len(v.simplePath))
	copy(result, v.simplePath)
	return result
}

// AddTracePoint 向车辆轨迹中添加一个记录点
// nodeID: 节点ID
// time: 时间戳
func (v *Vehicle) AddTracePoint(nodeID int64, time int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.trace[nodeID] = time
	v.lastTraceTime = time
}

// ShouldRecordTrace 判断当前是否应该记录轨迹
// time: 当前时间
// 基于时间采样规则，如果是起点、终点或满足时间间隔条件，则返回true
func (v *Vehicle) ShouldRecordTrace(time int) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// 只有车辆已进入系统(state>=3)时才考虑记录轨迹
	if v.state < 3 {
		return false
	}

	// 当前位置是起点或终点时必须记录
	if v.state >= 3 &&
		((v.pos != nil && v.pos.ID() == v.destination.ID()) || // 终点
			(v.state == 3 && v.origin != nil && v.pos == nil) || // 在缓冲区中的起点
			(v.state == 4 && v.pos != nil && v.pos.ID() == v.origin.ID())) { // 刚进入路网的起点
		return true
	}

	// 基于时间间隔决定是否记录 - 时间采样规则
	interval := v.traceInterval
	if interval <= 0 {
		// 根据配置更新轨迹记录间隔
		cfg := config.GetConfig()
		if cfg != nil && cfg.Trace.Enabled {
			// 使用配置中的轨迹记录间隔
			interval = cfg.Trace.TraceRecordInterval
			v.traceInterval = interval // 更新车辆的轨迹记录间隔
		} else {
			// 默认值
			interval = 10
			v.traceInterval = interval
		}
	}

	return time-v.lastTraceTime >= interval
}

// SetTraceInterval 设置轨迹记录间隔
func (v *Vehicle) SetTraceInterval(interval int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.traceInterval = interval
}

// CurrentPosition 返回车辆当前位置节点
func (v *Vehicle) CurrentPosition() graph.Node {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.pos
}

// InTime 返回车辆进入系统时间
func (v *Vehicle) InTime() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.inTime
}

// ClearTrace 清空车辆的轨迹数据
// 用于循环车辆开始新的行程前清空之前的轨迹记录
func (v *Vehicle) ClearTrace() {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 清空轨迹数据
	v.trace = make(map[int64]int)
	// 重置上次记录时间
	v.lastTraceTime = 0
}
