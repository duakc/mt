package services

import "sync"

var _ Registry = (*DefaultRegistry)(nil)

type DefaultRegistry struct {
	m *sync.Map
}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{m: &sync.Map{}}
}

func (r *DefaultRegistry) Store(k, v any) {
	r.m.Store(k, v)
}

func (r *DefaultRegistry) Load(k any) any {
	v, _ := r.m.Load(k)
	return v
}

func (r *DefaultRegistry) Clear() {
	r.m.Clear()
}
