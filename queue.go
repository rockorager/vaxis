package vaxis

import (
	"sync"
	"sync/atomic"
)

// queue provides an infinitely buffered channel
type queue[T any] struct {
	ch    chan T
	items []T
	mu    sync.Mutex
	busy  atomic.Bool
}

func newQueue[T any]() *queue[T] {
	q := &queue[T]{
		ch: make(chan T),
	}
	return q
}

func (q *queue[T]) Chan() chan T {
	return q.ch
}

func (q *queue[T]) push(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(q.items, item)
	if !q.busy.Load() {
		go q.process()
	}
}

func (q *queue[T]) pop() (T, bool) {
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

func (q *queue[T]) process() {
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
