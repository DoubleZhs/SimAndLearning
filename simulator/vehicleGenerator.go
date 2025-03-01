package simulator

import (
	"graphCA/element"
	"graphCA/utils"
	"sync"
	"sync/atomic"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func getNextVehicleID() int64 {
	return atomic.AddInt64(&numVehicleGenerated, 1)
}

func randomVelocity() int {
	return 1 + rand.Intn(2)
}

func randomAcceleration() int {
	return 1 + rand.Intn(3)
}

func randomSlowingProbability() float64 {
	return rand.Float64() / 2.0
}

func InitFixedVehicle(n int, g *simple.DirectedGraph, nodes []graph.Node, traceNodes []graph.Node) {
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			oCell := nodes[rand.Intn(len(nodes)-1)]
			minLength, maxLength := TripDistanceRange()
			allowedDCells := utils.AccessibleNodesWithinRange(g, oCell, minLength, maxLength)
			dCell := allowedDCells[rand.Intn(len(allowedDCells)-1)]

			var findO, findD int = 0, 0
			for _, node := range traceNodes {
				if node.ID() == oCell.ID() {
					findO = 1
				}
				if node.ID() == dCell.ID() {
					findD = 1
				}
			}
			if findO == 0 {
				traceNodes = append(traceNodes, oCell)
			}
			if findD == 0 {
				traceNodes = append(traceNodes, dCell)
			}

			vehicle := element.NewVehicle(getNextVehicleID(), randomVelocity(), randomAcceleration(), 1.0, randomSlowingProbability(), true) // ClosedVehicle = true
			vehicle.SetOD(g, oCell, dCell)

			path, _, err := utils.ShortestPath(g, oCell, dCell)
			if err != nil {
				panic(err)
			}
			vehicle.SetPath(path)

			vehicle.BufferIn(0)

			waitingVehiclesMutex.Lock()
			waitingVehicles[vehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()

			numVehiclesWaitingMutex.Lock()
			numVehiclesWaiting++
			numVehiclesWaitingMutex.Unlock()
		}()
	}
	wg.Wait()
}

func GenerateScheduleVehicle(simTime, n int, g *simple.DirectedGraph, nodes []graph.Node, traceNodes []graph.Node) {
	numVehiclesActiveMutex.Lock()
	defer numVehiclesActiveMutex.Unlock()

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			oCell := nodes[rand.Intn(len(nodes)-1)]
			minLength, maxLength := TripDistanceRange()
			allowedDCells := utils.AccessibleNodesWithinRange(g, oCell, minLength, maxLength)
			dCell := allowedDCells[rand.Intn(len(allowedDCells)-1)]

			var findO, findD int = 0, 0
			for _, node := range traceNodes {
				if node.ID() == oCell.ID() {
					findO = 1
				}
				if node.ID() == dCell.ID() {
					findD = 1
				}
			}
			if findO == 0 {
				traceNodes = append(traceNodes, oCell)
			}
			if findD == 0 {
				traceNodes = append(traceNodes, dCell)
			}

			vehicle := element.NewVehicle(getNextVehicleID(), randomVelocity(), randomAcceleration(), 1.0, randomSlowingProbability(), false)

			vehicle.SetOD(g, oCell, dCell)

			path, _, err := utils.ShortestPath(g, oCell, dCell)
			if err != nil {
				panic(err)
			}
			vehicle.SetPath(path)

			vehicle.BufferIn(simTime)

			waitingVehiclesMutex.Lock()
			waitingVehicles[vehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()

			numVehiclesWaitingMutex.Lock()
			numVehiclesWaiting++
			numVehiclesWaitingMutex.Unlock()
		}()
	}
	wg.Wait()
}
