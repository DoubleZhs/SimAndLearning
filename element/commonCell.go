package element

import (
	"container/list"
	"fmt"
	"sync"
)

// CommonCell 表示一个普通的单元格
type CommonCell struct {
	id         int64
	speedLimit int
	capacity   float64
	occupation float64
	container  map[*Vehicle]struct{}
	buffer     *list.List

	// 使用RWMutex替代Mutex以提高并发读取性能
	containerMux sync.RWMutex
	bufferMux    sync.RWMutex
}

// NewCommonCell 创建一个新的普通单元格
func NewCommonCell(id int64, speed int, capacity float64) *CommonCell {
	return &CommonCell{
		id:         id,
		speedLimit: speed,
		capacity:   capacity,
		occupation: 0,
		container:  make(map[*Vehicle]struct{}, 10), // 预分配更合适的初始容量
		buffer:     list.New(),
	}
}

// ID 返回单元格ID
func (cell *CommonCell) ID() int64 {
	return cell.id
}

// MaxSpeed 返回单元格的速度限制
func (cell *CommonCell) MaxSpeed() int {
	return cell.speedLimit
}

// Occupation 返回单元格的当前占用率
func (cell *CommonCell) Occupation() float64 {
	cell.containerMux.RLock()
	defer cell.containerMux.RUnlock()
	return cell.occupation
}

// Capacity 返回单元格的容量
func (cell *CommonCell) Capacity() float64 {
	return cell.capacity
}

// ListContainer 返回单元格中的所有车辆
func (cell *CommonCell) ListContainer() []*Vehicle {
	cell.containerMux.RLock()
	defer cell.containerMux.RUnlock()

	vehicles := make([]*Vehicle, 0, len(cell.container))
	for v := range cell.container {
		vehicles = append(vehicles, v)
	}
	return vehicles
}

// ListBuffer 返回单元格缓冲区中的所有车辆
func (cell *CommonCell) ListBuffer() []*Vehicle {
	cell.bufferMux.RLock()
	defer cell.bufferMux.RUnlock()

	vehicles := make([]*Vehicle, 0, cell.buffer.Len())
	for e := cell.buffer.Front(); e != nil; e = e.Next() {
		vehicles = append(vehicles, e.Value.(*Vehicle))
	}
	return vehicles
}

// ChangeToTrafficLightCell 将普通单元格转换为红绿灯单元格
func (cell *CommonCell) ChangeToTrafficLightCell(interval int, truePhaseInterval [2]int) *TrafficLightCell {
	return NewTrafficLightCell(cell.id, cell.speedLimit, cell.capacity, interval, truePhaseInterval)
}

// Loadable 检查单元格是否可以装载指定车辆
func (cell *CommonCell) Loadable(vehicle *Vehicle) bool {
	cell.containerMux.RLock()
	defer cell.containerMux.RUnlock()
	return cell.occupation+vehicle.occupy <= cell.capacity
}

// Load 将车辆装载到单元格中
func (cell *CommonCell) Load(vehicle *Vehicle) (bool, error) {
	cell.containerMux.Lock()
	defer cell.containerMux.Unlock()

	if cell.occupation+vehicle.occupy > cell.capacity {
		err := fmt.Errorf("cell %d current occupation %f, vehicle occupy %f, exceed capacity %f", cell.id, cell.occupation, vehicle.occupy, cell.capacity)
		return false, err
	}
	cell.container[vehicle] = struct{}{}
	cell.occupation += vehicle.occupy
	return true, nil
}

// Unload 从单元格中卸载车辆
func (cell *CommonCell) Unload(vehicle *Vehicle) (bool, error) {
	cell.containerMux.Lock()
	defer cell.containerMux.Unlock()

	if _, ok := cell.container[vehicle]; !ok {
		err := fmt.Errorf("cell %d does not contain vehicle %d", cell.id, vehicle.Index())
		return false, err
	}
	delete(cell.container, vehicle)
	cell.occupation -= vehicle.occupy
	return true, nil
}

// BufferLoad 将车辆添加到缓冲区
func (cell *CommonCell) BufferLoad(vehicle *Vehicle) bool {
	cell.bufferMux.Lock()
	defer cell.bufferMux.Unlock()

	cell.buffer.PushBack(vehicle)
	return true
}

// BufferUnload 从缓冲区中移除车辆
func (cell *CommonCell) BufferUnload(vehicle *Vehicle) (bool, error) {
	cell.bufferMux.Lock()
	defer cell.bufferMux.Unlock()

	for e := cell.buffer.Front(); e != nil; e = e.Next() {
		if e.Value.(*Vehicle) == vehicle {
			cell.buffer.Remove(e)
			return true, nil
		}
	}

	return false, fmt.Errorf("vehicle %d not found in buffer", vehicle.Index())
}
