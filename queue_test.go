package vaxis

import "testing"

func TestQueue(t *testing.T) {
	queue := NewQueue[int]()
	size := 10_000
	for i := 0; i < size; i += 1 {
		queue.Push(i)
	}
	for i, out := range queue.items {
		if i != out {
			t.Fatalf("event out of order: expected %d, got %d", i, out)
		}
	}
	// i := 0
	// for out := range queue.Chan() {
	// 	if i != out {
	// 	}
	// 	i += 1
	// }
}
