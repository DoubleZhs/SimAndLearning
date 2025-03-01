package simulator

import (
	"graphCA/element"
	"graphCA/recorder"
	"graphCA/utils"
	"sync"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func VehicleProcess(numWorkers, simTime int, g *simple.DirectedGraph, traceNodes []graph.Node) {
	checkCompletedVehicle(simTime, g, traceNodes)
	updateVehicleActiveStatus(numWorkers)
	updateVehiclePosition(numWorkers, simTime)
}

func checkCompletedVehicle(simTime int, g *simple.DirectedGraph, traceNodes []graph.Node) {
	if len(completedVehicles) == 0 {
		return
	}

	for vehicle := range completedVehicles {
		recorder.RecordVehicleData(vehicle)
		// recorder.RecordTraceData(vehicle)

		// closedVehicle 从当前位置重新进入系统
		if vehicle.Flag() {
			id, velocity, acceleration, slowingProb := vehicle.Index(), vehicle.Velocity(), vehicle.Acceleration(), vehicle.SlowingProb()

			newO := vehicle.Destination()
			minLength, maxLength := TripDistanceRange()
			allowedDCells := utils.AccessibleNodesWithinRange(g, newO, minLength, maxLength)
			newD := allowedDCells[rand.Intn(len(allowedDCells)-1)]

			var findO, findD int = 0, 0
			for _, node := range traceNodes {
				if node.ID() == newO.ID() {
					findO = 1
				}
				if node.ID() == newD.ID() {
					findD = 1
				}
			}
			if findO == 0 {
				traceNodes = append(traceNodes, newO)
			}
			if findD == 0 {
				traceNodes = append(traceNodes, newD)
			}

			newVehicle := element.NewVehicle(id, velocity, acceleration, 1.0, slowingProb, true)
			newVehicle.SetOD(g, newO, newD)

			// 路径
			path, _, err := utils.ShortestPath(g, newO, newD)
			if err != nil {
				panic(err)
			}
			newVehicle.SetPath(path)

			newVehicle.BufferIn(simTime)

			waitingVehiclesMutex.Lock()
			waitingVehicles[newVehicle] = struct{}{}
			waitingVehiclesMutex.Unlock()

			numVehiclesWaitingMutex.Lock()
			numVehiclesWaiting++
			numVehiclesWaitingMutex.Unlock()
		}
	}

	for v := range completedVehicles {
		delete(completedVehicles, v)
	}

}

func updateVehicleActiveStatus(numWorkers int) {
	if len(waitingVehicles) == 0 {
		return
	}

	vehicles := make([]*element.Vehicle, 0, len(waitingVehicles))
	for vehicle := range waitingVehicles {
		vehicles = append(vehicles, vehicle)
	}

	if len(vehicles) < numWorkers {
		numWorkers = len(vehicles)
	}

	vehiclesPerThread := len(vehicles) / numWorkers
	extraVehicles := len(vehicles) % numWorkers

	var wg sync.WaitGroup
	recordActivatedVehicle := make(chan *element.Vehicle, len(vehicles))

	start := 0
	for i := 0; i < numWorkers; i++ {
		end := start + vehiclesPerThread
		if i < extraVehicles {
			end++
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				vehicle := vehicles[j]
				ok := vehicle.UpdateActiveState()
				if ok {
					recordActivatedVehicle <- vehicle
				}
			}
		}(start, end)

		start = end
	}

	wg.Wait()
	close(recordActivatedVehicle)

	for vehicle := range recordActivatedVehicle {
		waitingVehiclesMutex.Lock()
		delete(waitingVehicles, vehicle)
		waitingVehiclesMutex.Unlock()

		numVehiclesWaitingMutex.Lock()
		numVehiclesWaiting--
		numVehiclesWaitingMutex.Unlock()

		activeVehiclesMutex.Lock()
		activeVehicles[vehicle] = struct{}{}
		activeVehiclesMutex.Unlock()

		numVehiclesActiveMutex.Lock()
		numVehiclesActive++
		numVehiclesActiveMutex.Unlock()
	}
}

func updateVehiclePosition(numWorkers, simTime int) {
	if len(activeVehicles) == 0 {
		return
	}

	vehicles := make([]*element.Vehicle, 0, len(activeVehicles))
	for vehicle := range activeVehicles {
		vehicles = append(vehicles, vehicle)
	}

	// 打乱vehicle顺序
	for i := len(vehicles) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		vehicles[i], vehicles[j] = vehicles[j], vehicles[i]
	}

	if len(vehicles) < numWorkers {
		numWorkers = len(vehicles)
	}

	vehiclesPerThread := len(vehicles) / numWorkers
	extraVehicles := len(vehicles) % numWorkers

	var wg sync.WaitGroup

	start := 0
	for i := 0; i < numWorkers; i++ {
		end := start + vehiclesPerThread
		if i < extraVehicles {
			end++
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				vehicle := vehicles[j]
				if vehicle.State() == 3 {
					vehicle.SystemIn()
				}
				if vehicle.State() == 4 {
					completed := vehicle.Move(simTime)
					if completed {
						activeVehiclesMutex.Lock()
						delete(activeVehicles, vehicle)
						activeVehiclesMutex.Unlock()

						numVehiclesActiveMutex.Lock()
						numVehiclesActive--
						numVehiclesActiveMutex.Unlock()

						completedVehiclesMutex.Lock()
						completedVehicles[vehicle] = struct{}{}
						completedVehiclesMutex.Unlock()

						numVehicleCompletedMutex.Lock()
						numVehicleCompleted++
						numVehicleCompletedMutex.Unlock()
					}
				}
			}
		}(start, end)

		start = end
	}

	wg.Wait()
}
