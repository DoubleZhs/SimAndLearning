package element

import (
	"errors"
	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

type Vehicle struct {
	index                    int64
	velocity, acceleration   int
	occupy                   float64
	slowingProb              float64
	tag                      float64
	flag                     bool
	state                    int
	graph                    *simple.DirectedGraph
	pos                      graph.Node
	origin, destination      graph.Node
	simplePath, residualPath []graph.Node
	pathlength               int
	inTime, outTime          int
	trace                    map[int64]int
	activiate                bool
}

func NewVehicle(index int64, velocity, acceleration int, occupy, slowingProb float64, flag bool) *Vehicle {
	return &Vehicle{
		index:        index,
		velocity:     velocity,
		acceleration: acceleration,
		occupy:       occupy,
		slowingProb:  slowingProb,
		tag:          rand.Float64(),
		flag:         flag,
		trace:        make(map[int64]int),
	}
}

func (v *Vehicle) Index() int64 {
	return v.index
}

func (v *Vehicle) Velocity() int {
	return v.velocity
}

func (v *Vehicle) Acceleration() int {
	return v.acceleration
}

func (v *Vehicle) SlowingProb() float64 {
	return v.slowingProb
}

func (v *Vehicle) State() int {
	return v.state
}

func (v *Vehicle) Flag() bool {
	return v.flag
}

func (v *Vehicle) Origin() graph.Node {
	return v.origin
}

func (v *Vehicle) Destination() graph.Node {
	return v.destination
}

func (v *Vehicle) Path() []graph.Node {
	return v.simplePath
}

func (v *Vehicle) PathLength() int {
	return v.pathlength
}

func (v *Vehicle) Trace() map[int64]int {
	return v.trace
}

func (v *Vehicle) Report() (int64, int, float64, int, int, float64, bool) {
	return v.index, v.acceleration, v.slowingProb, v.inTime, v.outTime, v.tag, v.flag
}

func (v *Vehicle) SetOD(g *simple.DirectedGraph, origin, destination graph.Node) (bool, error) {
	if origin.ID() == destination.ID() {
		return false, errors.New("origin and destination are the same")
	}
	v.graph = g
	v.origin = origin
	v.destination = destination
	v.state = 1
	return true, nil
}

func (v *Vehicle) SetPath(path []graph.Node) (bool, error) {
	if v.state != 1 {
		return false, errors.New("set origin and destination first")
	}
	if path[0] != v.origin {
		return false, errors.New("path does not start from origin")
	}
	if path[len(path)-1] != v.destination {
		return false, errors.New("path does not end at destination")
	}
	v.simplePath = path

	for _, node := range path {
		switch assertNode := node.(type) {
		case Cell:
			v.residualPath = append(v.residualPath, assertNode)
		case *Link:
			v.residualPath = append(v.residualPath, assertNode.Flat()...)
		default:
			panic("node is not a cell or link")
		}

	}
	v.pathlength = len(v.residualPath)
	v.state = 2
	return true, nil
}

func (v *Vehicle) BufferIn(inTime int) {
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

func (v *Vehicle) UpdateActiveState() bool {
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

func (v *Vehicle) SystemIn() {
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

func (v *Vehicle) SystemOut(time int) {
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

func (v *Vehicle) Move(time int) bool {
	for {
		v.accelerate()
		v.decelerate()
		v.randomSlowing()

		if v.velocity == 0 {
			return false
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

		v.pos.(Cell).Unload(v)
		targetCell.Load(v)
		v.pos = targetCell

		pathway := v.residualPath[:v.velocity]
		for _, checkNode := range v.simplePath {
			for _, node := range pathway {
				if checkNode.ID() == node.ID() {
					v.trace[checkNode.ID()] = time
				}
			}
		}

		v.residualPath = v.residualPath[v.velocity:]

		if len(v.residualPath) == 0 {
			v.SystemOut(time)
			return true
		}

		return false
	}
}

func (v *Vehicle) accelerate() {
	cell, ok := v.pos.(Cell)
	if !ok {
		panic("pos is not a cell")
	}
	v.velocity = min(v.velocity+v.acceleration, cell.MaxSpeed())
}

func (v *Vehicle) decelerate() {
	gap := v.calculateGap()
	v.velocity = min(v.velocity, gap)
}

func (v *Vehicle) calculateGap() int {
	gap := 0
	for i := 0; i < min(v.velocity, len(v.residualPath)); i++ {
		node := v.residualPath[i]
		cell, ok := node.(Cell)
		if !ok {
			panic("node is not a cell")
		}
		if !cell.Loadable(v) {
			break
		}

		// Check if the node is a cross node (inDegree > 1)
		toNodes := v.graph.To(node.ID())
		inDegree := 0
		for toNodes.Next() {
			inDegree++
		}
		if inDegree > 1 {
			// Calculate pass probability
			passProbability := 0.8
			if rand.Float64() > passProbability {
				return gap
			}
		}

		gap++
	}
	return gap
}

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
