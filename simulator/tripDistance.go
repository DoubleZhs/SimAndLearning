package simulator

import (
	"math"

	"math/rand/v2"
)

// 定义常量以提高代码可维护性
const (
	// 英里到公里的转换系数
	MILE_TO_KM float64 = 1.60934

	// 每个单元格的长度(米)
	CELL_LENGTH float64 = 7.5

	// 距离范围概率分布分界点
	PROB_SHORT_TRIP  float64 = 0.51 // 短途旅行概率(<=3.85英里)
	PROB_MEDIUM_TRIP float64 = 0.71 // 中途旅行概率(<=7.65英里)
	PROB_LONG_TRIP   float64 = 0.81 // 长途旅行概率(<=11.59英里)
	PROB_VERY_LONG   float64 = 0.92 // 很长旅行概率(<=19.68英里)
	PROB_EXTREME     float64 = 0.95 // 极长旅行概率(<=30英里)
	PROB_ULTRA       float64 = 0.99 // 超长旅行概率(<=70英里)

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

// TripDistanceLim 根据概率分布随机生成一个行程距离上限
// 返回换算成单元格数量的距离上限
// 注意：目前只使用了较短距离的概率分布
func TripDistanceLim() int {
	dice := rand.Float64()
	var lim float64

	switch {
	case dice <= PROB_SHORT_TRIP:
		lim = DIST_SHORT
	case dice <= PROB_MEDIUM_TRIP:
		lim = DIST_MEDIUM
	case dice <= PROB_LONG_TRIP:
		lim = DIST_LONG
	case dice <= PROB_VERY_LONG:
		lim = DIST_VERY_LONG
	case dice <= PROB_EXTREME:
		lim = DIST_EXTREME
	case dice <= PROB_ULTRA:
		lim = DIST_ULTRA
	default:
		lim = DIST_MAXIMUM
	}

	// 将英里转换为单元格数量
	return int(math.Round(lim * MILE_TO_KM * 1000 / CELL_LENGTH))
}

// TripDistanceRange 生成一个行程距离范围
// 返回换算成单元格数量的最小和最大距离
// 注意：目前实现中只使用了前几个概率区间
func TripDistanceRange() (int, int) {
	dice := rand.Float64()

	// 归一化 - 只考虑PROB_EXTREME以内的概率
	normalizedDice := dice / PROB_EXTREME

	var minDis, maxDis float64
	switch {
	case normalizedDice <= PROB_SHORT_TRIP/PROB_EXTREME:
		minDis, maxDis = DIST_VERY_SHORT, DIST_SHORT
	case normalizedDice <= PROB_MEDIUM_TRIP/PROB_EXTREME:
		minDis, maxDis = DIST_SHORT, DIST_MEDIUM
	case normalizedDice <= PROB_LONG_TRIP/PROB_EXTREME:
		minDis, maxDis = DIST_MEDIUM, DIST_LONG
	case normalizedDice <= PROB_VERY_LONG/PROB_EXTREME:
		minDis, maxDis = DIST_LONG, DIST_VERY_LONG
	default:
		minDis, maxDis = DIST_VERY_LONG, DIST_EXTREME
	}

	// 将英里转换为单元格数量
	minLength := int(math.Round(minDis * MILE_TO_KM * 1000 / CELL_LENGTH))
	maxLength := int(math.Round(maxDis * MILE_TO_KM * 1000 / CELL_LENGTH))

	return minLength, maxLength
}

// 注释掉的测试函数 - 保留以备将来参考
// func TripDistanceRange() (int, int) {
//     return 200, 6000
// }
