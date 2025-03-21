package simulator

import (
	"graphCA/config"
	"math"

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
	DEFAULT_PROB_SHORT_TRIP  float64 = 0.51 // 短途旅行概率(<=3.85英里)
	DEFAULT_PROB_MEDIUM_TRIP float64 = 0.71 // 中途旅行概率(<=7.65英里)
	DEFAULT_PROB_LONG_TRIP   float64 = 0.81 // 长途旅行概率(<=11.59英里)
	DEFAULT_PROB_VERY_LONG   float64 = 0.92 // 很长旅行概率(<=19.68英里)
	DEFAULT_PROB_EXTREME     float64 = 0.95 // 极长旅行概率(<=30英里)
	DEFAULT_PROB_ULTRA       float64 = 0.99 // 超长旅行概率(<=70英里)

	// 各个距离级别的上限(英里)
	DIST_VERY_SHORT float64 = 1.00
	DIST_SHORT      float64 = 3.85
	DIST_MEDIUM     float64 = 7.65
	DIST_LONG       float64 = 11.59
	DIST_VERY_LONG  float64 = 19.68
	DIST_EXTREME    float64 = 30.00
	DIST_ULTRA      float64 = 70.00
	DIST_MAXIMUM    float64 = 100.00
)

// 获取距离概率分布阈值，优先使用配置文件中的值，如果未配置则使用默认值
func getProbabilities() (float64, float64, float64, float64, float64, float64) {
	cfg := config.GetConfig()
	// 如果配置为nil或未设置相关配置，则使用默认值
	if cfg == nil {
		return DEFAULT_PROB_SHORT_TRIP, DEFAULT_PROB_MEDIUM_TRIP,
			DEFAULT_PROB_LONG_TRIP, DEFAULT_PROB_VERY_LONG,
			DEFAULT_PROB_EXTREME, DEFAULT_PROB_ULTRA
	}

	// 检查是否有自定义概率分布，如果所有值都为0则使用默认值
	if cfg.TripDistance.ProbShortTrip == 0 &&
		cfg.TripDistance.ProbMediumTrip == 0 &&
		cfg.TripDistance.ProbLongTrip == 0 &&
		cfg.TripDistance.ProbVeryLong == 0 &&
		cfg.TripDistance.ProbExtreme == 0 &&
		cfg.TripDistance.ProbUltra == 0 {
		return DEFAULT_PROB_SHORT_TRIP, DEFAULT_PROB_MEDIUM_TRIP,
			DEFAULT_PROB_LONG_TRIP, DEFAULT_PROB_VERY_LONG,
			DEFAULT_PROB_EXTREME, DEFAULT_PROB_ULTRA
	}

	// 使用配置文件中的值
	return cfg.TripDistance.ProbShortTrip, cfg.TripDistance.ProbMediumTrip,
		cfg.TripDistance.ProbLongTrip, cfg.TripDistance.ProbVeryLong,
		cfg.TripDistance.ProbExtreme, cfg.TripDistance.ProbUltra
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
// 注意：目前只使用了较短距离的概率分布
func TripDistanceLim() int {
	// 获取配置的概率分布
	probShort, probMedium, probLong, probVeryLong, probExtreme, probUltra := getProbabilities()
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
	case dice <= probExtreme:
		lim = DIST_EXTREME
	case dice <= probUltra:
		lim = DIST_ULTRA
	default:
		lim = DIST_MAXIMUM
	}

	// 应用最大距离倍数
	lim *= maxMult

	// 将英里转换为单元格数量
	return int(math.Round(lim * MILE_TO_KM * 1000 / CELL_LENGTH))
}

// TripDistanceRange 生成一个行程距离范围
// 返回换算成单元格数量的最小和最大距离
// 注意：目前实现中只使用了前几个概率区间
func TripDistanceRange() (int, int) {
	// 检查是否启用距离限制
	if !isDistanceLimitEnabled() {
		// 如果未启用距离限制，返回一个非常大的范围（实际上不限制）
		return 1, 1000000 // 几乎不限制距离
	}

	// 获取配置的概率分布
	probShort, probMedium, probLong, probVeryLong, probExtreme, _ := getProbabilities()
	minMult, maxMult := getDistanceMultipliers()

	dice := rand.Float64()

	// 归一化 - 只考虑probExtreme以内的概率
	normalizedDice := dice / probExtreme

	var minDis, maxDis float64
	switch {
	case normalizedDice <= probShort/probExtreme:
		minDis, maxDis = DIST_VERY_SHORT, DIST_SHORT
	case normalizedDice <= probMedium/probExtreme:
		minDis, maxDis = DIST_SHORT, DIST_MEDIUM
	case normalizedDice <= probLong/probExtreme:
		minDis, maxDis = DIST_MEDIUM, DIST_LONG
	case normalizedDice <= probVeryLong/probExtreme:
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

// 注释掉的测试函数 - 保留以备将来参考
// func TripDistanceRange() (int, int) {
//     return 200, 6000
// }
