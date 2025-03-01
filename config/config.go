package config

import (
	"encoding/json"
	"os"
)

// Config represents the root configuration structure
type Config struct {
	Simulation   SimulationConfig   `json:"simulation"`
	Logging      LoggingConfig      `json:"logging"`
	Demand       DemandConfig       `json:"demand"`
	Vehicle      VehicleConfig      `json:"vehicle"`
	TrafficLight TrafficLightConfig `json:"trafficLight"`
}

type SimulationConfig struct {
	OneDayTimeSteps   int `json:"oneDayTimeSteps"`
	SimDay            int `json:"simDay"`
	NumCell           int `json:"numCell"`
	LightIndexInterval int `json:"lightIndexInterval"`
}

type LoggingConfig struct {
	IntervalWriteToLog    int `json:"intervalWriteToLog"`
	IntervalWriteOtherData int `json:"intervalWriteOtherData"`
}

type DemandConfig struct {
	Multiplier        float64 `json:"multiplier"`
	FixedNum         float64 `json:"fixedNum"`
	DayRandomDisRange float64 `json:"dayRandomDisRange"`
	RandomDisRange    float64 `json:"randomDisRange"`
}

type VehicleConfig struct {
	NumClosedVehicle int `json:"numClosedVehicle"`
}

type TrafficLightChange struct {
	Day        int     `json:"day"`
	Multiplier float64 `json:"multiplier"`
}

type TrafficLightConfig struct {
	InitPhaseInterval int                  `json:"initPhaseInterval"`
	Changes          []TrafficLightChange `json:"changes"`
}

var globalConfig *Config

// LoadConfig loads configuration from the specified JSON file
func LoadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return err
	}

	globalConfig = config
	return nil
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	return globalConfig
} 