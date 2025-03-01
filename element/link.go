package element

import (
	"sync/atomic"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

var cellIndex int64 = 1000000

func getNextCellID() int64 {
	return atomic.AddInt64(&cellIndex, 1)
}

type Link struct {
	id         int64
	cells      []graph.Node
	numCells   int
	speedLimit int
	capacity   float64
}

func NewLink(id int64, numCells, speed int, capacity float64) *Link {
	if numCells < 2 {
		panic("numCells must be at least 2")
	}
	cells := make([]graph.Node, numCells)
	for i := 0; i < numCells; i++ {
		cells[i] = NewCommonCell(getNextCellID(), speed, float64(capacity))
	}
	return &Link{
		id:         id,
		cells:      cells,
		numCells:   numCells,
		speedLimit: speed,
		capacity:   capacity,
	}
}

func (l *Link) ID() int64 {
	return l.id
}

func (l *Link) Flat() []graph.Node {
	return l.cells
}

func (l *Link) Length() int {
	return l.numCells
}

func (l *Link) AddToGraph(g *simple.DirectedGraph) {
	for i := 0; i < len(l.cells)-1; i++ {
		g.SetEdge(simple.Edge{F: l.cells[i], T: l.cells[i+1]})
	}
}

func (l *Link) AddFromNode(g *simple.DirectedGraph, node graph.Node) {
	g.SetEdge(simple.Edge{F: node, T: l.cells[0]})
}

func (l *Link) AddToNode(g *simple.DirectedGraph, node graph.Node) {
	g.SetEdge(simple.Edge{F: l.cells[len(l.cells)-1], T: node})
}

func (l *Link) Report() (int, int, float64, int, float64) {
	allVehicle := make([]*Vehicle, 0)
	for _, cell := range l.cells {
		c := cell.(Cell)
		vehicles := c.ListContainer()
		allVehicle = append(allVehicle, vehicles...)
	}
	var totalSpeed float64 = 0
	for _, v := range allVehicle {
		totalSpeed += float64(v.velocity)
	}
	numVehicle := len(allVehicle)
	var averageSpeed float64 = 0
	if numVehicle == 0 {
		averageSpeed = 0
	} else {
		averageSpeed = totalSpeed / float64(numVehicle)
	}

	return l.numCells, l.speedLimit, l.capacity, numVehicle, averageSpeed

}
