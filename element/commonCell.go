package element

import (
	"container/list"
	"fmt"
	"sync"
)

type CommonCell struct {
	id         int64
	speedLimit int
	capacity   float64
	occupation float64
	container  map[*Vehicle]struct{}
	buffer     *list.List

	containerMux sync.Mutex
	bufferMux    sync.Mutex
}

func NewCommonCell(id int64, speed int, capacity float64) *CommonCell {
	return &CommonCell{
		id:         id,
		speedLimit: speed,
		capacity:   capacity,
		occupation: 0,
		container:  make(map[*Vehicle]struct{}),
		buffer:     list.New(),
	}
}

func (cell *CommonCell) ID() int64 {
	return cell.id
}

func (cell *CommonCell) MaxSpeed() int {
	return cell.speedLimit
}

func (cell *CommonCell) Occupation() float64 {
	return cell.occupation
}

func (cell *CommonCell) Capacity() float64 {
	return cell.capacity
}

func (cell *CommonCell) ListContainer() []*Vehicle {
	cell.containerMux.Lock()
	defer cell.containerMux.Unlock()
	vehicles := make([]*Vehicle, 0, len(cell.container))
	for v := range cell.container {
		vehicles = append(vehicles, v)
	}
	return vehicles
}

func (cell *CommonCell) ListBuffer() []*Vehicle {
	cell.bufferMux.Lock()
	defer cell.bufferMux.Unlock()
	vehicles := make([]*Vehicle, 0, cell.buffer.Len())
	for e := cell.buffer.Front(); e != nil; e = e.Next() {
		vehicles = append(vehicles, e.Value.(*Vehicle))
	}
	return vehicles
}

func (cell *CommonCell) ChangeToTrafficLightCell(interval int, truePhaseInterval [2]int) *TrafficLightCell {
	return NewTrafficLightCell(cell.id, cell.speedLimit, cell.capacity, interval, truePhaseInterval)
}

func (cell *CommonCell) Loadable(vehicle *Vehicle) bool {
	cell.containerMux.Lock()
	defer cell.containerMux.Unlock()
	return cell.occupation+vehicle.occupy <= cell.capacity
}

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

func (cell *CommonCell) BufferLoad(vehicle *Vehicle) bool {
	cell.bufferMux.Lock()
	defer cell.bufferMux.Unlock()
	cell.buffer.PushBack(vehicle)
	return true
}

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
