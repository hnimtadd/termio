package utils

func RotateOnce[T any](items []T) []T {
	tmp := items[0]
	copy(items, items[1:])
	items[len(items)-1] = tmp
	return items
}

func Rotate[T any](items []T, repeat int) []T {
	for range repeat {
		items = RotateOnce(items)
	}
	return items
}

func RotateOnceR[T any](items []T) []T {
	tmp := items[len(items)-1]
	copy(items[1:], items)
	items[0] = tmp
	return items
}

func RotateR[T any](items []T, repeat int) []T {
	for range repeat {
		items = RotateOnceR(items)
	}
	return items
}
