package services

import "sync"

var _ Registry = (*DefaultRegistry)(nil)

type DefaultRegistry struct {
	sealed chan struct{}

	m *sync.Map
}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{sealed: make(chan struct{}), m: &sync.Map{}}
}

func (r *DefaultRegistry) Store(k, v any) {
	select {
	case <-r.sealed:
		panic("store already sealed")
	default:
		r.m.Store(k, v)
	}
}

func (r *DefaultRegistry) Load(k any) any {
	v, _ := r.m.Load(k)
	return v
}

func (r *DefaultRegistry) Seal() {
	close(r.sealed)
}

func (r *DefaultRegistry) Clear() {
	r.m.Clear()
}
