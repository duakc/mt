package generic

import "sync"

type Pool[T any] struct {
	u *sync.Pool
}

func NewPool[T any](fn func() T) *Pool[T] {
	return &Pool[T]{
		u: &sync.Pool{
			New: func() any {
				return fn()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.u.Get().(T)
}

func (p *Pool[T]) Put(t T) {
	p.u.Put(t)
}
