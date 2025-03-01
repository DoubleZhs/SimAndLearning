package element

// TrafficLightCell 表示一个带有交通信号灯的单元格
type TrafficLightCell struct {
	CommonCell

	// 交通信号灯属性
	// phase表示当前相位状态(true为绿灯，false为红灯)
	// truePhaseInterval规定计数器属于该范围内时相位为true
	// interval表示一个完整周期的长度
	// count是当前周期内的计数
	phase             bool
	truePhaseInterval [2]int
	interval          int
	count             int
}

// NewTrafficLightCell 创建一个新的红绿灯单元格
func NewTrafficLightCell(id int64, speed int, capacity float64, interval int, truePhaseInterval [2]int) *TrafficLightCell {
	// 验证参数合法性
	if interval <= 0 {
		panic("interval must be positive")
	}
	if truePhaseInterval[0] < 0 || truePhaseInterval[1] <= truePhaseInterval[0] || truePhaseInterval[1] > interval {
		panic("invalid true phase interval")
	}

	return &TrafficLightCell{
		CommonCell:        *NewCommonCell(id, speed, capacity),
		truePhaseInterval: truePhaseInterval,
		interval:          interval,
		count:             0, // 初始化计数为0
	}
}

// Cycle 执行一个红绿灯周期
func (light *TrafficLightCell) Cycle() {
	light.count++
	if light.count > light.interval {
		light.count = 1
	}

	// 根据当前计数更新相位
	if light.count > light.truePhaseInterval[0] && light.count <= light.truePhaseInterval[1] {
		light.phase = true // 绿灯
	} else {
		light.phase = false // 红灯
	}
}

// Loadable 重写父类方法，考虑红绿灯状态
func (light *TrafficLightCell) Loadable(vehicle *Vehicle) bool {
	light.containerMux.RLock()
	defer light.containerMux.RUnlock()

	// 只有在绿灯状态且容量足够时才能通过
	return light.phase && (light.occupation+vehicle.occupy <= light.capacity)
}

// ChangeInterval 按指定倍数改变红绿灯周期
func (light *TrafficLightCell) ChangeInterval(mul float64) {
	if mul <= 0 {
		panic("multiplier must be positive")
	}

	// 按比例调整相关参数
	newInterval := int(float64(light.interval) * mul)
	newTruePhase := [2]int{
		int(float64(light.truePhaseInterval[0]) * mul),
		int(float64(light.truePhaseInterval[1]) * mul),
	}

	// 确保新的参数有效
	if newInterval <= 0 || newTruePhase[1] <= newTruePhase[0] {
		panic("invalid new interval parameters")
	}

	light.interval = newInterval
	light.truePhaseInterval = newTruePhase
	light.count = int(float64(light.count) * mul)

	// 确保计数在有效范围内
	if light.count > light.interval {
		light.count = light.interval
	}
	if light.count <= 0 {
		light.count = 1
	}
}

// SetCount 设置当前计数
func (light *TrafficLightCell) SetCount(count int) {
	if count <= 0 || count > light.interval {
		panic("count must be between 1 and interval")
	}
	light.count = count
}

// GetPhase 返回当前相位状态
func (light *TrafficLightCell) GetPhase() bool {
	return light.phase
}

// GetInterval 返回当前周期长度
func (light *TrafficLightCell) GetInterval() int {
	return light.interval
}

// GetTruePhaseInterval 返回绿灯相位区间
func (light *TrafficLightCell) GetTruePhaseInterval() [2]int {
	return light.truePhaseInterval
}
