package element

import "gonum.org/v1/gonum/graph"

type Cell interface {
	graph.Node
	MaxSpeed() int
	Occupation() float64
	Capacity() float64
	ListContainer() []*Vehicle
	ListBuffer() []*Vehicle
	Loadable(v *Vehicle) bool
	Load(v *Vehicle) (bool, error)
	Unload(v *Vehicle) (bool, error)
	BufferLoad(v *Vehicle) bool
	BufferUnload(v *Vehicle) (bool, error)
}
