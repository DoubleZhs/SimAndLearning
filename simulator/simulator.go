package simulator

import (
	"graphCA/element"
	"sync"
	"sync/atomic"

	"gonum.org/v1/gonum/graph"
)

var (
	activeVehicles    map[*element.Vehicle]struct{} = make(map[*element.Vehicle]struct{})
	waitingVehicles   map[*element.Vehicle]struct{} = make(map[*element.Vehicle]struct{})
	completedVehicles map[*element.Vehicle]struct{} = make(map[*element.Vehicle]struct{})

	activeVehiclesMutex    *sync.RWMutex = &sync.RWMutex{}
	waitingVehiclesMutex   *sync.RWMutex = &sync.RWMutex{}
	completedVehiclesMutex *sync.RWMutex = &sync.RWMutex{}

	numVehicleGenerated int64
	numVehiclesActive   int64
	numVehiclesWaiting  int64
	numVehicleCompleted int64
)

func GetVehiclesNum() (int64, int64, int64, int64) {
	return atomic.LoadInt64(&numVehicleGenerated),
		atomic.LoadInt64(&numVehiclesActive),
		atomic.LoadInt64(&numVehiclesWaiting),
		atomic.LoadInt64(&numVehicleCompleted)
}

func GetVehiclesOnRoad(nodes []graph.Node) map[*element.Vehicle]struct{} {
	vehiclesOnRoad := make(map[*element.Vehicle]struct{}, len(nodes))

	for _, node := range nodes {
		cell, ok := node.(element.Cell)
		if !ok {
			continue
		}

		container := cell.ListContainer()
		if len(container) > 0 {
			for _, vehicle := range container {
				vehiclesOnRoad[vehicle] = struct{}{}
			}
		}
	}
	return vehiclesOnRoad
}

func GetAverageSpeed_Density(vehiclesOnRoad map[*element.Vehicle]struct{}, numNodes int, avgLane float64) (float64, float64) {
	if len(vehiclesOnRoad) == 0 {
		return 0.0, 0.0
	}

	totalSpeed := 0.0
	for vehicle := range vehiclesOnRoad {
		totalSpeed += float64(vehicle.Velocity())
	}
	averageSpeed := totalSpeed / float64(len(vehiclesOnRoad))

	if numNodes <= 0 || avgLane <= 0 {
		return averageSpeed, 0.0
	}

	density := float64(len(vehiclesOnRoad)) / (float64(numNodes) * avgLane)

	return averageSpeed, density
}
