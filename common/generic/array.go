package generic

import "slices"

// LimitedArray only expose Add, Len, Snapshot api
// Can not be deleted or resorted
type LimitedArray[T any] struct {
	underlay []T
}

func (A *LimitedArray[T]) Add(vv ...T) {
	A.underlay = append(A.underlay, vv...)
}

func (A *LimitedArray[T]) Len() int {
	return len(A.underlay)
}

func (A *LimitedArray[T]) Snapshot() []T {
	return slices.Clone(A.underlay)
}
