package recorder

import (
	"fmt"
	"graphCA/element"
	"strconv"
	"sync"
)

type linkData struct {
	SimTime      int
	numCell      int
	speedLim     int
	capacity     float64
	numVehicle   int
	averageSpeed float64
	density      float64
}

var (
	linkDataCache map[*element.Link][]linkData = make(map[*element.Link][]linkData)
	linkDataMutex sync.Mutex                   = sync.Mutex{}
)

func RecordLinkData(simTime int, links []*element.Link) {
	linkDataMutex.Lock()
	defer linkDataMutex.Unlock()
	for _, link := range links {
		linkDataCache[link] = append(linkDataCache[link], getLinkData(simTime, link))
	}
}

func getLinkData(simTime int, link *element.Link) linkData {
	numCell, speedLim, capacity, numVehicle, averageSpeed := link.Report()
	density := float64(numVehicle) / (float64(numCell) * capacity)

	return linkData{
		SimTime:      simTime,
		numCell:      numCell,
		speedLim:     speedLim,
		capacity:     capacity,
		numVehicle:   numVehicle,
		averageSpeed: averageSpeed,
		density:      density,
	}

}

func InitLinkDataCSV(filename string) {
	header := []string{
		"ID", "SimTime0", "SimTime1", "NumCell", "SpeedLim", "Capacity", "avgNumVehicle", "avgAverageSpeed", "avgDensity",
	}
	initializeCSV(filename, header)
}

func WriteToLinkDataCSV(filename string, links []*element.Link) {
	linkDataMutex.Lock()
	defer linkDataMutex.Unlock()
	if len(linkDataCache) == 0 {
		return
	}
	var linksData [][]string
	for link, data := range linkDataCache {
		simTime0, simTime1, numCell, speedLim, capacity, avgNumVehicle, avgAverageSpeed, avgDensity := avgLinkData(data)
		linksData = append(linksData, formatLinkDataOutput(link.ID(), simTime0, simTime1, numCell, speedLim, capacity, avgNumVehicle, avgAverageSpeed, avgDensity))
	}

	appendToCSV(filename, linksData)
	linkDataCache = make(map[*element.Link][]linkData)
}

func avgLinkData(data []linkData) (int, int, int, int, float64, float64, float64, float64) {
	if len(data) == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0
	}

	numCell := data[0].numCell
	speedLim := data[0].speedLim
	capacity := data[0].capacity

	simTime0 := data[0].SimTime
	simTime1 := data[0].SimTime
	var totalNumVehicle int
	var totalAverageSpeed float64
	var totalDensity float64

	for _, d := range data {
		if d.SimTime < simTime0 {
			simTime0 = d.SimTime
		}
		if d.SimTime > simTime1 {
			simTime1 = d.SimTime
		}
		totalNumVehicle += d.numVehicle
		totalAverageSpeed += d.averageSpeed
		totalDensity += d.density
	}

	count := float64(len(data))
	avgNumVehicle := float64(totalNumVehicle) / count
	avgAverageSpeed := totalAverageSpeed / count
	avgDensity := totalDensity / count

	return simTime0, simTime1, numCell, speedLim, capacity, avgNumVehicle, avgAverageSpeed, avgDensity
}

func formatLinkDataOutput(linkId int64, simTime0, simTime1, numCell, speedLim int, capacity, avgNumVehicle, avgAverageSpeed, avgDensity float64) []string {
	return []string{
		strconv.FormatInt(linkId, 10),
		strconv.Itoa(simTime0),
		strconv.Itoa(simTime1),
		strconv.Itoa(numCell),
		strconv.Itoa(speedLim),
		fmt.Sprintf("%.4f", capacity),
		fmt.Sprintf("%.4f", avgNumVehicle),
		fmt.Sprintf("%.4f", avgAverageSpeed),
		fmt.Sprintf("%.4f", avgDensity),
	}

}
