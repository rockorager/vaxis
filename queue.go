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

// NewQueue creates a new Queue, with items T. Queue is essentially an
// infinitely buffered channel. Items can be accessed via Chan, and inserted via
// Push
func NewQueue[T any]() *Queue[T] {
	q := &Queue[T]{
		ch: make(chan T),
	}
	return q
}

// Access the items in the Queue
func (q *Queue[T]) Chan() chan T {
	return q.ch
}

// Push adds an item to the Queue
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
