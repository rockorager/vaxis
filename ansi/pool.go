package ansi

import (
	"sync"
)

// A pool is a generic wrapper around a sync.pool.
type pool[T any] struct {
	pool sync.Pool
}

// Create a new pool which will use the fn to create new instances of T
func newPool[T any](fn func() T) pool[T] {
	return pool[T]{
		pool: sync.Pool{New: func() interface{} { return fn() }},
	}
}

// Get a new T
func (p *pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put a T back in the pool
func (p *pool[T]) Put(x T) {
	p.pool.Put(x)
}
