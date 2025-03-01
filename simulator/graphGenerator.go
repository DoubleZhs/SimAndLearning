package simulator

import (
	"graphCA/element"
	"math"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func CreateCycleGraph(cellNum int, trafficLightInterval int, initInterval int) (*simple.DirectedGraph, map[int64]graph.Node, map[int64]*element.TrafficLightCell) {
	g := simple.NewDirectedGraph()

	nodes := make(map[int64]graph.Node)
	lights := make(map[int64]*element.TrafficLightCell)

	trafficLightRatioCount := 0
	for i := 0; i < cellNum; i++ {
		if i%trafficLightInterval == 0 {
			greenRatio := 0.0
			if trafficLightRatioCount%2 == 0 {
				greenRatio = 0.3
			} else {
				greenRatio = 0.7
			}
			light := element.NewTrafficLightCell(int64(i), 5, 1.0, initInterval, [2]int{0, int(math.Round(float64(initInterval) * greenRatio))})
			light.SetCount(rand.Intn(initInterval))
			g.AddNode(light)
			nodes[int64(i)] = light
			lights[int64(i)] = light
			trafficLightRatioCount++
		} else {
			cell := element.NewCommonCell(int64(i), 5, 1.0)
			g.AddNode(cell)
			nodes[int64(i)] = cell
		}
	}

	for i := 0; i < cellNum-1; i++ {
		g.SetEdge(simple.Edge{F: nodes[int64(i)], T: nodes[int64((i + 1))]})
	}
	g.SetEdge(simple.Edge{F: nodes[int64(cellNum-1)], T: nodes[int64(0)]})

	return g, nodes, lights
}
