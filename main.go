package main

import (
	"fmt"
	"runtime"
	"simAndLearning/config"
	"simAndLearning/element"
	"simAndLearning/log"
	"simAndLearning/recorder"
	"simAndLearning/simulator"
	"simAndLearning/utils"
	"time"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func main() {
	// Load configuration file
	if err := config.LoadConfig("config/config.json"); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	cfg := config.GetConfig()

	// Generate unique timestamp for file naming
	initTime := time.Now().Format("2006010215040506")

	// Initialize resources
	_, dataFiles := initializeResources(cfg, initTime)
	defer func() {
		log.CloseLog()
	}()

	// Initialize simulation environment
	g, nodes, lights, avgLane := initializeSimulationEnvironment(cfg, initTime)
	numNodes := len(nodes)

	// Initialize system state
	sysState := simulator.NewSystemState()
	var demand []float64

	// Initialize vehicles
	simulator.InitFixedVehicle(cfg.Vehicle.NumClosedVehicle, g, nodes)

	// Start simulation
	log.WriteLog("----------------------------------Simulation Start----------------------------------")
	runSimulation(cfg, g, nodes, lights, numNodes, avgLane, sysState, &demand, dataFiles)

	// Complete simulation, write final data
	simulator.FinishSimulation(dataFiles)

	log.WriteLog("---------------------------------- Completed ----------------------------------")
}

// Initialize system resources
func initializeResources(cfg *config.Config, initTime string) (string, map[string]string) {
	// Initialize logging
	logFile := fmt.Sprintf("./log/%s_%d.log", initTime, cfg.Vehicle.NumClosedVehicle)
	log.InitLog(logFile)
	log.LogEnvironment()

	// Record simulation parameters
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

	// Record network parameters
	log.WriteLog(fmt.Sprintf("Graph Type: %s", cfg.Graph.GraphType))
	if cfg.Graph.GraphType == "cycle" {
		log.WriteLog(fmt.Sprintf("Cycle Graph Cell Count: %d", cfg.Graph.CycleGraph.NumCell))
		log.WriteLog(fmt.Sprintf("Cycle Graph Traffic Light Interval: %d", cfg.Graph.CycleGraph.LightIndexInterval))
	} else if cfg.Graph.GraphType == "starRing" {
		log.WriteLog(fmt.Sprintf("StarRing Graph Ring Cells Per Direction: %d", cfg.Graph.StarRingGraph.RingCellsPerDirection))
		log.WriteLog(fmt.Sprintf("StarRing Graph Star Cells Per Direction: %d", cfg.Graph.StarRingGraph.StarCellsPerDirection))
	}

	log.WriteLog(fmt.Sprintf("Concurrent Volume in Vehicle Process: %d", runtime.GOMAXPROCS(0)))

	// Initialize CSV data files
	systemDataFile := fmt.Sprintf("./data/%s_%d_SystemData.csv", initTime, cfg.Vehicle.NumClosedVehicle)
	vehicleDataFile := fmt.Sprintf("./data/%s_%d_VehicleData.csv", initTime, cfg.Vehicle.NumClosedVehicle)

	recorder.InitSystemDataCSV(systemDataFile)
	recorder.InitVehicleDataCSV(vehicleDataFile)

	dataFiles := map[string]string{
		"system":  systemDataFile,
		"vehicle": vehicleDataFile,
	}

	return logFile, dataFiles
}

// Initialize simulation environment
func initializeSimulationEnvironment(cfg *config.Config, initTime string) (*simple.DirectedGraph, []graph.Node, map[int64]*element.TrafficLightCell, float64) {
	var g *simple.DirectedGraph
	var nodesMap map[int64]graph.Node
	var lights map[int64]*element.TrafficLightCell
	var err error

	// Use the passed timestamp for file naming
	graphFilePath := fmt.Sprintf("./data/%s_%d_Graph.json", initTime, cfg.Vehicle.NumClosedVehicle)

	// Select and create graph based on configuration
	switch cfg.Graph.GraphType {
	case "cycle":
		// Create and save cycle graph
		g, nodesMap, lights, err = simulator.SaveCycleGraph(
			cfg.Graph.CycleGraph.NumCell,
			cfg.Graph.CycleGraph.LightIndexInterval,
			cfg.TrafficLight.InitPhaseInterval,
			graphFilePath,
		)
		if err != nil {
			log.WriteLog(fmt.Sprintf("Failed to save cycle graph: %v", err))
		} else {
			log.WriteLog(fmt.Sprintf("Cycle graph saved to: %s", graphFilePath))
		}
	case "starRing":
		// Create and save star-ring hybrid graph
		g, nodesMap, lights, err = simulator.SaveStarRingGraph(
			cfg.Graph.StarRingGraph.RingCellsPerDirection,
			cfg.Graph.StarRingGraph.StarCellsPerDirection,
			cfg.TrafficLight.InitPhaseInterval,
			graphFilePath,
		)
		if err != nil {
			log.WriteLog(fmt.Sprintf("Failed to save star-ring graph: %v", err))
		} else {
			log.WriteLog(fmt.Sprintf("Star-ring graph saved to: %s", graphFilePath))
		}
	default:
		// Default to cycle graph
		log.WriteLog(fmt.Sprintf("Unknown graph type: %s, using default cycle graph", cfg.Graph.GraphType))
		g, nodesMap, lights, err = simulator.SaveCycleGraph(
			cfg.Graph.CycleGraph.NumCell,
			cfg.Graph.CycleGraph.LightIndexInterval,
			cfg.TrafficLight.InitPhaseInterval,
			graphFilePath,
		)
		if err != nil {
			log.WriteLog(fmt.Sprintf("Failed to save cycle graph: %v", err))
		} else {
			log.WriteLog(fmt.Sprintf("Cycle graph saved to: %s", graphFilePath))
		}
	}

	numNodes := len(nodesMap)
	log.WriteLog(fmt.Sprintf("Graph Type: %s", cfg.Graph.GraphType))
	log.WriteLog(fmt.Sprintf("Total Nodes: %d", numNodes))
	log.WriteLog(fmt.Sprintf("Traffic Lights Count: %d", len(lights)))

	// Convert map to slice for easier processing
	nodes := make([]graph.Node, 0, numNodes)
	// Calculate total lanes
	var allLane float64
	for _, node := range nodesMap {
		nodes = append(nodes, node)
		allLane += node.(element.Cell).Capacity()
	}
	// Calculate average lane count
	avgLane := allLane / float64(numNodes)
	log.WriteLog(fmt.Sprintf("Average Lanes: %.2f", avgLane))

	// Check if graph is strongly connected
	gConnect := utils.IsStronglyConnected(g)
	log.WriteLog(fmt.Sprintf("Graph Connectivity: %v", gConnect))

	// For starRing graphs, use more detailed connectivity check
	if cfg.Graph.GraphType == "starRing" && !gConnect {
		isConnected, problems := simulator.VerifyStarRingGraphConnectivity(g)
		log.WriteLog(fmt.Sprintf("StarRing Graph Detailed Connectivity Check: %v", isConnected))
		if !isConnected && len(problems) > 0 {
			log.WriteLog("Connectivity Issues:")
			for _, problem := range problems {
				log.WriteLog(fmt.Sprintf("- %s", problem))
			}
		}
	}

	return g, nodes, lights, avgLane
}

// Run simulation
func runSimulation(cfg *config.Config, g *simple.DirectedGraph, nodes []graph.Node, lights map[int64]*element.TrafficLightCell,
	numNodes int, avgLane float64, sysState *simulator.SystemState, demand *[]float64,
	dataFiles map[string]string) {

	simDaySteps := cfg.Simulation.SimDay * cfg.Simulation.OneDayTimeSteps

	// Main simulation loop
	for timeStep := 0; timeStep < simDaySteps; timeStep++ {
		timeOfDay := timeStep % cfg.Simulation.OneDayTimeSteps
		currentDay := timeStep/cfg.Simulation.OneDayTimeSteps + 1

		// Update demand distribution at the start of each day
		if timeOfDay == 0 {
			*demand = simulator.AdjustDemand(
				cfg.Demand.Multiplier,
				cfg.Demand.FixedNum,
				cfg.Demand.DayRandomDisRange,
			)
		}

		// Check for traffic light cycle changes
		for _, change := range cfg.TrafficLight.Changes {
			if currentDay == change.Day && timeOfDay == 0 {
				for _, light := range lights {
					light.ChangeInterval(change.Multiplier)
				}
				log.WriteLog(fmt.Sprintf("TrafficLight Interval Changed: Multiplier - %.2f", change.Multiplier))
			}
		}

		// Generate and process vehicles
		generateNum := simulator.GetGenerateVehicleCount(timeOfDay, *demand, cfg.Demand.RandomDisRange)
		simulator.GenerateScheduleVehicle(timeStep, generateNum, g, nodes)

		// Traffic light cycle
		for _, light := range lights {
			light.Cycle()
		}

		// Process vehicle movement
		simulator.VehicleProcess(runtime.GOMAXPROCS(0), timeStep, g)

		// Update system state
		sysState.Update(nodes, numNodes, avgLane)
		sysState.RecordData(timeStep)

		// Log at intervals
		if timeOfDay%cfg.Logging.IntervalWriteToLog == 0 {
			sysState.LogStatus(currentDay, timeOfDay)
		}

		// Write system and vehicle data at intervals
		if timeOfDay%cfg.Logging.IntervalWriteOtherData == 0 {
			simulator.WriteData(dataFiles)
		}
	}

	// Perform final data write at simulation end to ensure all data is written
	simulator.WriteData(dataFiles)
}
