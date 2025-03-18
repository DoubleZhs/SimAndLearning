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
	Trace        TraceConfig        `json:"trace"`
}

type SimulationConfig struct {
	OneDayTimeSteps    int `json:"oneDayTimeSteps"`
	SimDay             int `json:"simDay"`
	NumCell            int `json:"numCell"`
	LightIndexInterval int `json:"lightIndexInterval"`
}

type LoggingConfig struct {
	IntervalWriteToLog     int `json:"intervalWriteToLog"`
	IntervalWriteOtherData int `json:"intervalWriteOtherData"`
}

type DemandConfig struct {
	Multiplier        float64 `json:"multiplier"`
	FixedNum          float64 `json:"fixedNum"`
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
	Changes           []TrafficLightChange `json:"changes"`
}

// TraceConfig 保存轨迹记录相关的配置项
type TraceConfig struct {
	// 是否启用轨迹记录功能
	Enabled bool `json:"enabled"`
	// 检查点数量（每辆车记录的位置点数量）
	// 如果不设置或设置为0，系统将根据路径长度自动确定合适的检查点数量
	CheckpointInterval int `json:"checkpointInterval"`
	// 轨迹数据写入间隔（时间步），独立于其他数据的写入间隔
	WriteInterval int `json:"writeInterval"`
	// 内存管理相关配置
	// 最大缓存条目数，超过此值触发写入
	MaxCacheSize int `json:"maxCacheSize"`
	// 内存使用阈值(MB)，超过此值触发写入
	MemoryThreshold int `json:"memoryThreshold"`
	// 是否启用内存监控
	EnableMemoryMonitor bool `json:"enableMemoryMonitor"`
	// 内存监控检查间隔(记录次数)
	MemoryCheckInterval int `json:"memoryCheckInterval"`
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

	// 设置轨迹记录的默认值（如果未在配置文件中指定）
	if config.Trace.CheckpointInterval <= 0 {
		config.Trace.CheckpointInterval = 10 // 默认每辆车记录10个点
	}

	// 为新添加的配置项设置默认值
	if config.Trace.WriteInterval <= 0 {
		config.Trace.WriteInterval = config.Logging.IntervalWriteOtherData // 默认与其他数据写入间隔相同
	}

	if config.Trace.MaxCacheSize <= 0 {
		config.Trace.MaxCacheSize = 10000 // 默认缓存10000条记录
	}

	if config.Trace.MemoryThreshold <= 0 {
		config.Trace.MemoryThreshold = 500 // 默认500MB
	}

	if config.Trace.MemoryCheckInterval <= 0 {
		config.Trace.MemoryCheckInterval = 1000 // 默认每1000次记录检查一次
	}

	// 默认启用内存监控
	if !config.Trace.EnableMemoryMonitor {
		config.Trace.EnableMemoryMonitor = true
	}

	globalConfig = config
	return nil
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	return globalConfig
}
