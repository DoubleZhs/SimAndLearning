package config

import (
	"encoding/json"
	"os"
)

// Config 保存所有配置项的顶级结构
type Config struct {
	Simulation   SimulationConfig   `json:"simulation"`
	Logging      LoggingConfig      `json:"logging"`
	Demand       DemandConfig       `json:"demand"`
	Vehicle      VehicleConfig      `json:"vehicle"`
	TrafficLight TrafficLightConfig `json:"trafficLight"`
	Graph        GraphConfig        `json:"graph"`
	Path         PathConfig         `json:"path"`
	TripDistance TripDistanceConfig `json:"tripDistance"`
}

// SimulationConfig 保存模拟相关的配置项
type SimulationConfig struct {
	OneDayTimeSteps int `json:"oneDayTimeSteps"`
	SimDay          int `json:"simDay"`
}

// GraphConfig 保存路网相关的配置项
type GraphConfig struct {
	// 路网类型: "cycle" - 环形路网, "starRing" - 星形环形混合路网
	GraphType string `json:"graphType"`

	// 环形路网参数
	CycleGraph struct {
		NumCell            int `json:"numCell"`
		LightIndexInterval int `json:"lightIndexInterval"`
	} `json:"cycleGraph"`

	// 星形环形混合路网参数
	StarRingGraph struct {
		RingCellsPerDirection int `json:"ringCellsPerDirection"`
		StarCellsPerDirection int `json:"starCellsPerDirection"`
	} `json:"starRingGraph"`
}

// LoggingConfig 保存日志记录相关的配置项
type LoggingConfig struct {
	IntervalWriteToLog     int `json:"intervalWriteToLog"`
	IntervalWriteOtherData int `json:"intervalWriteOtherData"`
}

// DemandConfig 保存需求生成相关的配置项
type DemandConfig struct {
	Multiplier        float64 `json:"multiplier"`
	FixedNum          float64 `json:"fixedNum"`
	DayRandomDisRange float64 `json:"dayRandomDisRange"`
	RandomDisRange    float64 `json:"randomDisRange"`
}

// VehicleConfig 保存车辆相关的配置项
type VehicleConfig struct {
	NumClosedVehicle int `json:"numClosedVehicle"`
	TraceInterval    int `json:"traceInterval"`
}

// TrafficLightChange 表示流量灯变化的配置
type TrafficLightChange struct {
	Day        int     `json:"day"`
	Multiplier float64 `json:"multiplier"`
}

// TrafficLightConfig 保存交通信号灯相关的配置项
type TrafficLightConfig struct {
	InitPhaseInterval int                  `json:"initPhaseInterval"`
	Changes           []TrafficLightChange `json:"changes"`
}

// PathConfig 管理车辆路径选择相关的配置
type PathConfig struct {
	// 路径选择方法: "shortest" - 最短路径, "random" - 随机路径, "kShortest" - k条最短路径中选择
	PathMethod string `json:"pathMethod"`

	// k最短路径相关参数
	KShortest struct {
		// 计算的最短路径数量
		K int `json:"k"`

		// 路径选择策略: "random" - 随机选择, "weighted" - 加权选择
		SelectionStrategy string `json:"selectionStrategy"`

		// 路径长度权重因子，值越大对短路径的偏好越强（仅在weighted策略下有效）
		LengthWeightFactor float64 `json:"lengthWeightFactor"`
	} `json:"kShortest"`
}

// TripDistanceConfig 管理车辆出行距离相关的配置
type TripDistanceConfig struct {
	// 是否启用距离限制, true-启用距离范围限制, false-随机选择任意可达目的地
	EnableDistanceLimit bool `json:"enableDistanceLimit"`

	// 自定义概率分布比例（仅在enableDistanceLimit=true时有效）
	// 如果未指定或所有值均为0，则使用默认概率分布
	// 注：所有概率现已放缩到DIST_EXTREME(30英里)以内
	ProbShortTrip  float64 `json:"probShortTrip"`  // 短途旅行概率(<=3.85英里)
	ProbMediumTrip float64 `json:"probMediumTrip"` // 中途旅行概率(<=7.65英里)
	ProbLongTrip   float64 `json:"probLongTrip"`   // 长途旅行概率(<=11.59英里)
	ProbVeryLong   float64 `json:"probVeryLong"`   // 很长旅行概率(<=19.68英里)
	ProbExtreme    float64 `json:"probExtreme"`    // 极长旅行概率(<=30英里)，会自动放缩为1.0

	// 最小距离倍数（相对于最小默认距离）
	MinDistMultiplier float64 `json:"minDistMultiplier"`

	// 最大距离倍数（相对于最大默认距离）
	MaxDistMultiplier float64 `json:"maxDistMultiplier"`
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

	// 设置路网配置的默认值
	if config.Graph.GraphType == "" {
		config.Graph.GraphType = "cycle" // 默认使用环形路网
	}

	// 设置环形路网参数的默认值
	if config.Graph.CycleGraph.NumCell <= 0 {
		config.Graph.CycleGraph.NumCell = 8000 // 默认单元格数量
	}
	if config.Graph.CycleGraph.LightIndexInterval <= 0 {
		config.Graph.CycleGraph.LightIndexInterval = 800 // 默认红绿灯间隔
	}

	// 设置星形环形路网参数的默认值
	if config.Graph.StarRingGraph.RingCellsPerDirection <= 0 {
		config.Graph.StarRingGraph.RingCellsPerDirection = 600 // 默认环形路径单元格数
	}
	if config.Graph.StarRingGraph.StarCellsPerDirection <= 0 {
		config.Graph.StarRingGraph.StarCellsPerDirection = 400 // 默认星形路径单元格数
	}

	// 设置路径配置的默认值
	if config.Path.PathMethod == "" {
		config.Path.PathMethod = "shortest" // 默认使用最短路径
	}

	if config.Path.KShortest.K <= 0 {
		config.Path.KShortest.K = 3 // 默认计算3条最短路径
	}

	if config.Path.KShortest.SelectionStrategy == "" {
		config.Path.KShortest.SelectionStrategy = "random" // 默认使用随机选择策略
	}

	if config.Path.KShortest.LengthWeightFactor <= 0 {
		config.Path.KShortest.LengthWeightFactor = 1.0 // 默认权重因子
	}

	// 设置出行距离配置的默认值
	// 默认启用距离限制
	if config.TripDistance.MinDistMultiplier <= 0 {
		config.TripDistance.MinDistMultiplier = 1.0 // 默认不缩放最小距离
	}

	if config.TripDistance.MaxDistMultiplier <= 0 {
		config.TripDistance.MaxDistMultiplier = 1.0 // 默认不缩放最大距离
	}

	// 设置车辆轨迹记录间隔的默认值
	if config.Vehicle.TraceInterval <= 0 {
		config.Vehicle.TraceInterval = 1 // 默认每个时间步记录
	}

	globalConfig = config
	return nil
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	return globalConfig
}
