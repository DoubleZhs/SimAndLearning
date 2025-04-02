package simulator

import (
	"math"
	"simAndLearning/config"

	"math/rand/v2"

	"gonum.org/v1/gonum/graph"
)

// 定义常量以提高代码可维护性
const (
	// 英里到公里的转换系数
	MILE_TO_KM float64 = 1.60934

	// 每个单元格的长度(米)
	CELL_LENGTH float64 = 7.5

	// 距离范围概率分布分界点（默认值）
	// 注：所有概率都放缩到DIST_EXTREME以内
	DEFAULT_PROB_SHORT_TRIP  float64 = 0.54 // 短途旅行概率(<=3.85英里)，原0.51放缩
	DEFAULT_PROB_MEDIUM_TRIP float64 = 0.75 // 中途旅行概率(<=7.65英里)，原0.71放缩
	DEFAULT_PROB_LONG_TRIP   float64 = 0.85 // 长途旅行概率(<=11.59英里)，原0.81放缩
	DEFAULT_PROB_VERY_LONG   float64 = 0.97 // 很长旅行概率(<=19.68英里)，原0.92放缩
	DEFAULT_PROB_EXTREME     float64 = 1.00 // 极长旅行概率(<=30英里)，原0.95放缩为1.0

	// 各个距离级别的上限(英里)
	DIST_VERY_SHORT float64 = 1.01 // 修改为1.01英里，确保最短出行距离在1英里以上
	DIST_SHORT      float64 = 3.85
	DIST_MEDIUM     float64 = 7.65
	DIST_LONG       float64 = 11.59
	DIST_VERY_LONG  float64 = 19.68
	DIST_EXTREME    float64 = 30.00
	// 不再使用ULTRA和MAXIMUM
	// DIST_ULTRA      float64 = 70.00
	// DIST_MAXIMUM    float64 = 100.00
)

// 获取距离概率分布阈值，优先使用配置文件中的值，如果未配置则使用默认值
func getProbabilities() (float64, float64, float64, float64, float64) {
	cfg := config.GetConfig()
	// 如果配置为nil或未设置相关配置，则使用默认值
	if cfg == nil {
		return DEFAULT_PROB_SHORT_TRIP, DEFAULT_PROB_MEDIUM_TRIP,
			DEFAULT_PROB_LONG_TRIP, DEFAULT_PROB_VERY_LONG,
			DEFAULT_PROB_EXTREME
	}

	// 检查是否有自定义概率分布，如果所有值都为0则使用默认值
	if cfg.TripDistance.ProbShortTrip == 0 &&
		cfg.TripDistance.ProbMediumTrip == 0 &&
		cfg.TripDistance.ProbLongTrip == 0 &&
		cfg.TripDistance.ProbVeryLong == 0 &&
		cfg.TripDistance.ProbExtreme == 0 {
		return DEFAULT_PROB_SHORT_TRIP, DEFAULT_PROB_MEDIUM_TRIP,
			DEFAULT_PROB_LONG_TRIP, DEFAULT_PROB_VERY_LONG,
			DEFAULT_PROB_EXTREME
	}

	// 使用配置文件中的值并放缩到EXTREME以内
	// 如果配置的极限概率不是1，则进行放缩
	probExtreme := cfg.TripDistance.ProbExtreme
	if probExtreme < 1.0 && probExtreme > 0 {
		// 放缩因子
		scaleFactor := 1.0 / probExtreme

		return math.Min(cfg.TripDistance.ProbShortTrip*scaleFactor, 1.0),
			math.Min(cfg.TripDistance.ProbMediumTrip*scaleFactor, 1.0),
			math.Min(cfg.TripDistance.ProbLongTrip*scaleFactor, 1.0),
			math.Min(cfg.TripDistance.ProbVeryLong*scaleFactor, 1.0),
			1.0 // EXTREME总是为1.0
	}

	// 如果probExtreme已经是1或无效值，则直接使用原值，但确保EXTREME为1
	return cfg.TripDistance.ProbShortTrip,
		cfg.TripDistance.ProbMediumTrip,
		cfg.TripDistance.ProbLongTrip,
		cfg.TripDistance.ProbVeryLong,
		1.0 // EXTREME总是为1.0
}

// 获取距离倍数，优先使用配置文件中的值，如果未配置则使用默认值1.0
func getDistanceMultipliers() (float64, float64) {
	cfg := config.GetConfig()
	// 如果配置为nil或未设置相关配置，则使用默认值
	if cfg == nil {
		return 1.0, 1.0
	}

	minMult := cfg.TripDistance.MinDistMultiplier
	if minMult <= 0 {
		minMult = 1.0
	}

	maxMult := cfg.TripDistance.MaxDistMultiplier
	if maxMult <= 0 {
		maxMult = 1.0
	}

	return minMult, maxMult
}

// 检查是否启用距离限制
func isDistanceLimitEnabled() bool {
	cfg := config.GetConfig()
	// 如果配置为nil，则默认启用距离限制
	if cfg == nil {
		return true
	}

	return cfg.TripDistance.EnableDistanceLimit
}

// TripDistanceLim 根据概率分布随机生成一个行程距离上限
// 返回换算成单元格数量的距离上限
func TripDistanceLim() int {
	// 获取配置的概率分布
	probShort, probMedium, probLong, probVeryLong, _ := getProbabilities()
	_, maxMult := getDistanceMultipliers()

	dice := rand.Float64()
	var lim float64

	switch {
	case dice <= probShort:
		lim = DIST_SHORT
	case dice <= probMedium:
		lim = DIST_MEDIUM
	case dice <= probLong:
		lim = DIST_LONG
	case dice <= probVeryLong:
		lim = DIST_VERY_LONG
	default:
		lim = DIST_EXTREME
	}

	// 应用最大距离倍数
	lim *= maxMult

	// 将英里转换为单元格数量
	return int(math.Round(lim * MILE_TO_KM * 1000 / CELL_LENGTH))
}

// TripDistanceRange 生成一个行程距离范围
// 返回换算成单元格数量的最小和最大距离
func TripDistanceRange() (int, int) {
	// 检查是否启用距离限制
	if !isDistanceLimitEnabled() {
		// 如果未启用距离限制，返回一个非常大的范围（实际上不限制）
		// 但确保最小距离在1英里以上
		minLength := int(math.Round(DIST_VERY_SHORT * MILE_TO_KM * 1000 / CELL_LENGTH))
		return minLength, 1000000 // 最小距离设为DIST_VERY_SHORT，最大距离几乎不限制
	}

	// 获取配置的概率分布
	probShort, probMedium, probLong, probVeryLong, _ := getProbabilities()
	minMult, maxMult := getDistanceMultipliers()

	dice := rand.Float64()

	var minDis, maxDis float64
	switch {
	case dice <= probShort:
		minDis, maxDis = DIST_VERY_SHORT, DIST_SHORT
	case dice <= probMedium:
		minDis, maxDis = DIST_SHORT, DIST_MEDIUM
	case dice <= probLong:
		minDis, maxDis = DIST_MEDIUM, DIST_LONG
	case dice <= probVeryLong:
		minDis, maxDis = DIST_LONG, DIST_VERY_LONG
	default:
		minDis, maxDis = DIST_VERY_LONG, DIST_EXTREME
	}

	// 应用距离倍数
	minDis *= minMult
	maxDis *= maxMult

	// 将英里转换为单元格数量
	minLength := int(math.Round(minDis * MILE_TO_KM * 1000 / CELL_LENGTH))
	maxLength := int(math.Round(maxDis * MILE_TO_KM * 1000 / CELL_LENGTH))

	return minLength, maxLength
}

// GetRandomDestination 从所有节点中随机选择目的地
// 当不启用距离限制时使用
func GetRandomDestination(nodes []graph.Node, excludeNode graph.Node) graph.Node {
	// 创建一个临时列表，排除起点
	availableNodes := make([]graph.Node, 0, len(nodes)-1)
	for _, node := range nodes {
		if node.ID() != excludeNode.ID() {
			availableNodes = append(availableNodes, node)
		}
	}

	// 如果没有可用节点，返回nil
	if len(availableNodes) == 0 {
		return nil
	}

	// 随机选择一个节点作为目的地
	return availableNodes[rand.IntN(len(availableNodes))]
}
