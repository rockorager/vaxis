package vaxis

import (
	"sync"
	"sync/atomic"
)

// Queue provides an infinitely buffered channel
type Queue[T any] struct {
	ch     chan T
	items  []T
	mu     sync.Mutex
	busy   atomic.Bool
	closed bool
}

func NewQueue[T any]() *Queue[T] {
	q := &Queue[T]{
		ch: make(chan T),
	}
	return q
}

func (q *Queue[T]) Chan() chan T {
	return q.ch
}

func (q *Queue[T]) Push(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}

	q.items = append(q.items, item)
	if !q.busy.Load() {
		go q.process()
	}
}

func (q *Queue[T]) pop() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var item T
	switch len(q.items) {
	case 0:
		return item, false
	case 1:
		item = q.items[0]
		q.items = make([]T, 0)
	default:
		item = q.items[0]
		q.items = q.items[1:]
	}
	return item, true
}

func (q *Queue[T]) process() {
	q.busy.Store(true)
	defer q.busy.Store(false)

	for {
		item, ok := q.pop()
		if !ok {
			return
		}
		q.ch <- item
	}
}
