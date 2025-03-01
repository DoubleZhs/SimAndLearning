package simulator

import (
	"encoding/csv"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"math/rand/v2"
)

// 交通需求原始数据，从CSV文件中读取
var rawDemand []float64 = readDemandCSV()

// AdjustDemand 调整原始需求数据
//
// 参数:
//   - A: 需求乘数
//   - B: 需求偏移量
//   - randomDis: 随机波动范围 (0-1)
//
// 返回:
//   - []float64: 调整后的需求数据列表
//
// 公式: adjusted = (raw * A + B) * (1 + random_factor)
// 其中 random_factor 在 [-randomDis, +randomDis] 范围内
func AdjustDemand(A, B, randomDis float64) []float64 {
	// 验证参数
	if randomDis < 0 || randomDis > 1 {
		log.Printf("Warning: randomDis should be between 0 and 1, got %f", randomDis)
		randomDis = math.Max(0, math.Min(1, randomDis)) // 限制在 0-1 范围内
	}

	adjustedDemand := make([]float64, len(rawDemand))

	// 生成随机因子，范围在 [1-randomDis, 1+randomDis]
	randomFactor := 1 + (rand.Float64()*2*randomDis - randomDis)

	// 为每个时段调整需求
	for i, d := range rawDemand {
		adjustedDemand[i] = (d*A + B) * randomFactor

		// 确保调整后的需求非负
		if adjustedDemand[i] < 0 {
			adjustedDemand[i] = 0
		}
	}

	return adjustedDemand
}

// GetGenerateVehicleCount 根据时间和需求列表计算应生成的车辆数量
//
// 参数:
//   - timeOfDay: 一天中的时段索引
//   - dayDemandList: 一天各时段的需求列表
//   - randomDis: 随机波动范围 (0-1)
//
// 返回:
//   - int: 应生成的车辆数量
//
// 算法:
//  1. 应用随机波动到基础需求值
//  2. 取整数部分作为基础车辆数
//  3. 剩余小数部分作为生成额外车辆的概率
func GetGenerateVehicleCount(timeOfDay int, dayDemandList []float64, randomDis float64) int {
	// 验证参数
	if timeOfDay < 0 || timeOfDay >= len(dayDemandList) {
		log.Printf("Warning: timeOfDay %d is out of range (0-%d)", timeOfDay, len(dayDemandList)-1)
		// 确保timeOfDay在有效范围内
		if timeOfDay < 0 {
			timeOfDay = 0
		} else if timeOfDay >= len(dayDemandList) {
			timeOfDay = len(dayDemandList) - 1
		}
	}

	if randomDis < 0 || randomDis > 1 {
		log.Printf("Warning: randomDis should be between 0 and 1, got %f", randomDis)
		randomDis = math.Max(0, math.Min(1, randomDis)) // 限制在 0-1 范围内
	}

	// 生成随机因子，范围在 [1-randomDis, 1+randomDis]
	randomFactor := 1 + (rand.Float64()*2*randomDis - randomDis)
	baseDemand := dayDemandList[timeOfDay] * randomFactor

	// 确保需求非负
	if baseDemand < 0 {
		baseDemand = 0
	}

	// 取整数部分作为基础车辆数
	baseN := math.Floor(baseDemand)

	// 使用小数部分作为生成额外车辆的概率
	var randomN float64
	randomDice := rand.Float64()
	if randomDice < baseDemand-baseN {
		randomN = 1
	} else {
		randomN = 0
	}

	return int(baseN + randomN)
}

// readDemandCSV 从CSV文件读取交通需求分布数据
//
// 返回:
//   - []float64: 交通需求数据列表
//
// 文件格式:
//
//	第一行为标题
//	之后每行包含时段ID和对应的需求值
func readDemandCSV() []float64 {
	// 设置文件路径
	filename := "./resources/DemandTimeDistribution_Smoothed.csv"

	// 使用相对路径查找文件
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// 尝试查找当前目录和上级目录
		dirs := []string{".", "..", "../..", "../../.."}
		found := false

		for _, dir := range dirs {
			path := filepath.Join(dir, "resources", "DemandTimeDistribution_Smoothed.csv")
			if _, err := os.Stat(path); err == nil {
				filename = path
				found = true
				break
			}
		}

		if !found {
			log.Printf("Warning: Demand data file not found at %s, using empty demand list", filename)
			return []float64{}
		}
	}

	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open demand file: %s", err)
		return []float64{} // 返回空列表而不是崩溃
	}
	defer file.Close()

	// 读取CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Failed to read CSV: %s", err)
		return []float64{} // 返回空列表而不是崩溃
	}

	// 检查文件格式
	if len(records) < 2 {
		log.Printf("Demand data file has insufficient data")
		return []float64{}
	}

	// 解析需求数据
	demand := make([]float64, 0, len(records)-1)
	for i, record := range records[1:] {
		// 确保记录包含足够的字段
		if len(record) < 2 {
			log.Printf("Warning: Record at line %d has insufficient fields", i+2)
			continue
		}

		// 解析概率值
		pro, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			log.Printf("Warning: Failed to parse probability at line %d: %s", i+2, err)
			continue
		}

		demand = append(demand, pro)
	}

	log.Printf("Loaded %d demand data points from %s", len(demand), filename)
	return demand
}
