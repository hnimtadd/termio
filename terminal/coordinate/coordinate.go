package coordinate

type Point[T comparable] struct {
	X T
	Y T
}

func NewPoint[T comparable](x, y T) Point[T] {
	return Point[T]{X: x, Y: y}
}
