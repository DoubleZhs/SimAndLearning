package simulator

import (
	"fmt"
	"simAndLearning/element"
	"simAndLearning/log"
	"simAndLearning/recorder"
	"sync"

	"gonum.org/v1/gonum/graph"
)

// SystemState 缓存并管理系统状态信息
// 包括车辆数量、平均速度、密度等关键指标
type SystemState struct {
	numVehicleGenerated int64
	numVehiclesActive   int64
	numVehiclesWaiting  int64
	numVehicleCompleted int64
	vehiclesOnRoad      map[*element.Vehicle]struct{}
	averageSpeed        float64
	density             float64
	mu                  sync.RWMutex // 保护并发访问
}

// NewSystemState 创建一个新的系统状态对象
func NewSystemState() *SystemState {
	return &SystemState{
		vehiclesOnRoad: make(map[*element.Vehicle]struct{}),
	}
}

// Update 更新系统状态
// 从模拟器中获取最新的车辆数量、分布和速度信息
func (s *SystemState) Update(nodes []graph.Node, numNodes int, avgLane float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.numVehicleGenerated, s.numVehiclesActive, s.numVehiclesWaiting, s.numVehicleCompleted = GetVehiclesNum()
	s.vehiclesOnRoad = GetVehiclesOnRoad(nodes)
	s.averageSpeed, s.density = GetAverageSpeed_Density(s.vehiclesOnRoad, numNodes, avgLane)
}

// RecordData 记录当前系统状态数据
// 将数据传递给recorder进行存储
func (s *SystemState) RecordData(timeStep int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recorder.RecordSystemData(timeStep, s.numVehicleGenerated, s.numVehiclesActive,
		s.numVehiclesWaiting, s.numVehicleCompleted, s.averageSpeed, s.density)
}

// LogStatus 输出系统状态日志
// 格式化并打印当前系统状态信息
func (s *SystemState) LogStatus(currentDay int, timeOfDay int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.WriteLog(fmt.Sprintf("Day: %d, TimeOfDay: %v, AvgSpeed: %.2f, Density: %.2f, Generated: %d, Active: %d, OnRoad: %d, Waiting: %d, Completed: %d",
		currentDay, log.ConvertTimeStepToTime(timeOfDay), s.averageSpeed, s.density,
		s.numVehicleGenerated, s.numVehiclesActive, len(s.vehiclesOnRoad),
		s.numVehiclesWaiting, s.numVehicleCompleted))
}

// GetVehiclesOnRoadCount 返回当前道路上的车辆数量
func (s *SystemState) GetVehiclesOnRoadCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.vehiclesOnRoad)
}

// GetAverageSpeed 返回当前系统的平均车速
func (s *SystemState) GetAverageSpeed() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.averageSpeed
}

// GetDensity 返回当前系统的车辆密度
func (s *SystemState) GetDensity() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.density
}

// GetVehicleCounts 返回各类车辆计数
// 返回值依次为: 生成的车辆总数、活动车辆数、等待车辆数、已完成车辆数
func (s *SystemState) GetVehicleCounts() (int64, int64, int64, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.numVehicleGenerated, s.numVehiclesActive, s.numVehiclesWaiting, s.numVehicleCompleted
}
