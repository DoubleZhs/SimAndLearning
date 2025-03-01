package simulator

import (
	"graphCA/element"
	"sync"

	"gonum.org/v1/gonum/graph"
)

var (
	activeVehicles    map[*element.Vehicle]struct{} = make(map[*element.Vehicle]struct{})
	waitingVehicles   map[*element.Vehicle]struct{} = make(map[*element.Vehicle]struct{})
	completedVehicles map[*element.Vehicle]struct{} = make(map[*element.Vehicle]struct{})

	activeVehiclesMutex    *sync.Mutex = &sync.Mutex{}
	waitingVehiclesMutex   *sync.Mutex = &sync.Mutex{}
	completedVehiclesMutex *sync.Mutex = &sync.Mutex{}

	numVehicleGenerated int64
	numVehiclesActive   int64
	numVehiclesWaiting  int64
	numVehicleCompleted int64

	numVehicleGeneratedMutex *sync.Mutex = &sync.Mutex{}
	numVehiclesActiveMutex   *sync.Mutex = &sync.Mutex{}
	numVehiclesWaitingMutex  *sync.Mutex = &sync.Mutex{}
	numVehicleCompletedMutex *sync.Mutex = &sync.Mutex{}
)

func GetVehiclesNum() (int64, int64, int64, int64) {
	numVehicleGeneratedMutex.Lock()
	numVehiclesActiveMutex.Lock()
	numVehiclesWaitingMutex.Lock()
	numVehicleCompletedMutex.Lock()
	defer numVehicleGeneratedMutex.Unlock()
	defer numVehiclesActiveMutex.Unlock()
	defer numVehiclesWaitingMutex.Unlock()
	defer numVehicleCompletedMutex.Unlock()

	return numVehicleGenerated, numVehiclesActive, numVehiclesWaiting, numVehicleCompleted
}

func GetVehiclesOnRoad(nodes []graph.Node) map[*element.Vehicle]struct{} {
	vehiclesOnRaod := make(map[*element.Vehicle]struct{})
	for _, node := range nodes {
		container := node.(element.Cell).ListContainer()
		if len(container) > 0 {
			for _, vehicle := range container {
				vehiclesOnRaod[vehicle] = struct{}{}
			}
		}
	}
	return vehiclesOnRaod
}

// func GetVehiclesOnRoad() map[*element.Vehicle]struct{} {
// 	activeVehiclesMutex.Lock()
// 	defer activeVehiclesMutex.Unlock()

// 	vehiclesOnRaod := make(map[*element.Vehicle]struct{})
// 	for vehicle := range activeVehicles {
// 		if vehicle.State() == 4 {
// 			vehiclesOnRaod[vehicle] = struct{}{}
// 		}
// 	}

// 	return vehiclesOnRaod
// }

func GetAverageSpeed_Density(vehiclesOnRaod map[*element.Vehicle]struct{}, numNodes int, avgLane float64) (float64, float64) {
	totalSpeed := 0.0
	for vehicle := range vehiclesOnRaod {
		totalSpeed += float64(vehicle.Velocity())
	}
	averageSpeed := totalSpeed / float64(len(vehiclesOnRaod))

	density := float64(len(vehiclesOnRaod)) / (float64(numNodes) * avgLane)

	return averageSpeed, density
}
