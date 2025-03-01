package element

import (
	"sync"
	"sync/atomic"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// 共享的原子计数器，用于生成唯一的单元格ID
var cellIndex int64 = 1000000

// getNextCellID 生成下一个唯一的单元格ID
func getNextCellID() int64 {
	return atomic.AddInt64(&cellIndex, 1)
}

// Link 表示连接两个节点的链路，包含多个单元格
type Link struct {
	id         int64        // 链路ID
	cells      []graph.Node // 链路包含的单元格
	numCells   int          // 单元格数量
	speedLimit int          // 速度限制
	capacity   float64      // 每个单元格的容量
	mu         sync.RWMutex // 用于保护并发访问
}

// NewLink 创建一个新的链路
func NewLink(id int64, numCells, speed int, capacity float64) *Link {
	if numCells < 2 {
		panic("numCells must be at least 2")
	}

	// 预分配容量以提高性能
	cells := make([]graph.Node, numCells)
	for i := 0; i < numCells; i++ {
		cells[i] = NewCommonCell(getNextCellID(), speed, capacity)
	}

	return &Link{
		id:         id,
		cells:      cells,
		numCells:   numCells,
		speedLimit: speed,
		capacity:   capacity,
	}
}

// ID 返回链路ID
func (l *Link) ID() int64 {
	return l.id
}

// Flat 返回链路包含的所有单元格
func (l *Link) Flat() []graph.Node {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回副本以避免外部修改
	result := make([]graph.Node, len(l.cells))
	copy(result, l.cells)
	return result
}

// Length 返回链路长度（单元格数量）
func (l *Link) Length() int {
	return l.numCells
}

// AddToGraph 将链路添加到图中
func (l *Link) AddToGraph(g *simple.DirectedGraph) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for i := 0; i < len(l.cells)-1; i++ {
		g.SetEdge(simple.Edge{F: l.cells[i], T: l.cells[i+1]})
	}
}

// AddFromNode 将node连接到链路的起点
func (l *Link) AddFromNode(g *simple.DirectedGraph, node graph.Node) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	g.SetEdge(simple.Edge{F: node, T: l.cells[0]})
}

// AddToNode 将链路的终点连接到node
func (l *Link) AddToNode(g *simple.DirectedGraph, node graph.Node) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	g.SetEdge(simple.Edge{F: l.cells[len(l.cells)-1], T: node})
}

// Report 报告链路的状态信息
// 返回：单元格数量，速度限制，容量，车辆数量，平均速度
func (l *Link) Report() (int, int, float64, int, float64) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 收集所有车辆
	allVehicle := make([]*Vehicle, 0, l.numCells*2) // 预估每个单元格平均2辆车
	for _, cell := range l.cells {
		c := cell.(Cell)
		vehicles := c.ListContainer()
		allVehicle = append(allVehicle, vehicles...)
	}

	// 计算平均速度
	var totalSpeed float64 = 0
	for _, v := range allVehicle {
		totalSpeed += float64(v.velocity)
	}

	numVehicle := len(allVehicle)
	var averageSpeed float64 = 0
	if numVehicle > 0 {
		averageSpeed = totalSpeed / float64(numVehicle)
	}

	return l.numCells, l.speedLimit, l.capacity, numVehicle, averageSpeed
}

// GetCell 返回链路中指定索引的单元格
func (l *Link) GetCell(index int) (Cell, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index < 0 || index >= len(l.cells) {
		return nil, false
	}

	return l.cells[index].(Cell), true
}
