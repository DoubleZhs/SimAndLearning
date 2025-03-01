package log

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

var logFile *os.File

func LogEnvironment() {
	log.Println("------------------------Environment-----------------------")
	log.Printf("Operating System: %s\n", runtime.GOOS)
	log.Printf("Architecture: %s\n", runtime.GOARCH)
	log.Printf("Number of CPUs: %d\n", runtime.NumCPU())
	log.Printf("Compiler Version: %s\n", runtime.Version())
}

func LogSimParameters(oneDayTimeSteps int, demandMultiplier, demandFixedNum, demandRandomDisRange float64, numClosedVehicle, simDay, initTrafficLightPhaseInterval, trafficLightChangeDay1 int, trafficLightMul1 float64, trafficLightChangeDay2 int, trafficLightMul2 float64) {
	log.Println("--------------------Simulation Setting--------------------")
	log.Printf("One Day Time Steps: %d\n", oneDayTimeSteps)
	log.Printf("One Cell Length: %.2f\n", 7.5)
	log.Printf("Demand Multiplier: %.2f\n", demandMultiplier)
	log.Printf("Demand Fixed Num: %.2f\n", demandFixedNum)
	log.Printf("Demand Random DisRange: %.2f\n", demandRandomDisRange)
	log.Printf("Closed Vehicle Num: %d\n", numClosedVehicle)
	log.Printf("Simulation Days: %d\n", simDay)
	log.Printf("Init Traffic Light True Phase Interval: %d\n", initTrafficLightPhaseInterval)
	log.Printf("Traffic Light Change Day1: %d\n", trafficLightChangeDay1)
	log.Printf("TrafficLight Mul1: %f\n", trafficLightMul1)
	log.Printf("Traffic Light Change Day2: %d\n", trafficLightChangeDay2)
	log.Printf("TrafficLight Mul2: %f\n", trafficLightMul2)
	log.Println("------------------------------------------------------------")
}

func ConvertTimeStepToTime(time int) string {
	hours := (time / 2400) % 24
	minutes := (time % 2400) / 40
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

func InitLog(logFilename string) {
	var err error
	logFile, err = os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %s", err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)
}

func WriteLog(message string) {
	log.Println(message)
}

func CloseLog() {
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			log.Fatalf("Failed to close log file: %s", err)
		}
	}
}
