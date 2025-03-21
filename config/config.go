package config

import (
	"encoding/json"
	"os"
	"reflect"
)

// Config represents the root configuration structure
type Config struct {
	Simulation   SimulationConfig   `json:"simulation"`
	Logging      LoggingConfig      `json:"logging"`
	Demand       DemandConfig       `json:"demand"`
	Vehicle      VehicleConfig      `json:"vehicle"`
	TrafficLight TrafficLightConfig `json:"trafficLight"`
	Trace        TraceConfig        `json:"trace"`
	Graph        GraphConfig        `json:"graph"`
	Path         PathConfig         `json:"path"`
	TripDistance TripDistanceConfig `json:"tripDistance"`
}

type SimulationConfig struct {
	OneDayTimeSteps int `json:"oneDayTimeSteps"`
	SimDay          int `json:"simDay"`
}

// GraphConfig 管理路网相关的配置
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
	// 轨迹记录间隔（车辆移动多少步记录一次位置）
	// 如果不设置或设置为0，系统将根据路径长度自动确定合适的记录间隔
	TraceRecordInterval int `json:"traceRecordInterval"`
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
	// 是否按天拆分轨迹数据
	SplitByDay bool `json:"splitByDay"`
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
	ProbShortTrip  float64 `json:"probShortTrip"`  // 短途旅行概率(<=3.85英里)
	ProbMediumTrip float64 `json:"probMediumTrip"` // 中途旅行概率(<=7.65英里)
	ProbLongTrip   float64 `json:"probLongTrip"`   // 长途旅行概率(<=11.59英里)
	ProbVeryLong   float64 `json:"probVeryLong"`   // 很长旅行概率(<=19.68英里)
	ProbExtreme    float64 `json:"probExtreme"`    // 极长旅行概率(<=30英里)
	ProbUltra      float64 `json:"probUltra"`      // 超长旅行概率(<=70英里)

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

	// 设置轨迹记录的默认值（如果未在配置文件中指定）
	if config.Trace.TraceRecordInterval <= 0 {
		config.Trace.TraceRecordInterval = 10 // 默认每辆车记录10个点
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

	// 设置路网配置的默认值
	if config.Graph.GraphType == "" {
		config.Graph.GraphType = "cycle" // 默认使用环形路网
	}

	// 如果配置文件中没有路网参数，但simulation中有旧的参数格式，则进行兼容处理
	// 这段代码用于向后兼容，可在未来版本中移除
	if config.Graph.CycleGraph.NumCell == 0 && config.Graph.CycleGraph.LightIndexInterval == 0 {
		// 尝试查找在SimulationConfig中的旧格式参数
		field := reflect.ValueOf(config.Simulation).FieldByName("NumCell")
		if field.IsValid() {
			config.Graph.CycleGraph.NumCell = int(field.Int())
		}

		field = reflect.ValueOf(config.Simulation).FieldByName("LightIndexInterval")
		if field.IsValid() {
			config.Graph.CycleGraph.LightIndexInterval = int(field.Int())
		}
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
		config.Graph.StarRingGraph.RingCellsPerDirection = 100 // 默认环形路径单元格数
	}
	if config.Graph.StarRingGraph.StarCellsPerDirection <= 0 {
		config.Graph.StarRingGraph.StarCellsPerDirection = 80 // 默认星形路径单元格数
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

	globalConfig = config
	return nil
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	return globalConfig
}
