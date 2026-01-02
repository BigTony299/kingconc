package kingconc

import "sync"

// NewQueue creates a new thread-safe queue that can be used concurrently
// from multiple goroutines. The queue follows FIFO (First In, First Out) ordering.
func NewQueue[T any]() Queue[T] {
	return Queue[T]{
		nextChan: make(chan T),
		inner:    make([]T, 20),
	}
}

// Queue is a thread-safe concurrent queue implementation that supports
// multiple goroutines safely pushing and popping elements simultaneously.
// It uses a read-write mutex to allow concurrent reads (Peek, Count) while
// ensuring exclusive access for writes (Push, Pop).
type Queue[T any] struct {
	mu sync.RWMutex

	nextChan chan T

	count int

	inner []T

	headIndex int
}

// Push adds an element to the back of the queue.
// If there are goroutines blocked on Next() waiting for items, the element will be
// immediately delivered to one of them via an internal channel. Otherwise, the element
// is stored in the internal queue following FIFO ordering.
//
// This operation is thread-safe and can be called concurrently from multiple goroutines.
// Push never blocks and always completes immediately.
func (a *Queue[T]) Push(t T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	select {
	case a.nextChan <- t:
		// direct transfer

	default:
		if a.count >= len(a.inner) {
			currLen := len(a.inner)

			var newLen int
			if currLen == 0 {
				newLen = 10
			} else {
				newLen = 2 * currLen
			}

			newInner := make([]T, newLen)

			for i := range currLen {
				j := (i + a.headIndex) % currLen

				newInner[i] = a.inner[j]
			}

			a.inner = newInner
			a.headIndex = 0
		}

		next := (a.headIndex + a.count) % len(a.inner)
		a.inner[next] = t
		a.count += 1
	}
}

// Pop removes and returns the element from the front of the queue.
// Returns Some(value) if the queue contains elements, or None() if empty.
// This operation is thread-safe and can be called concurrently from multiple goroutines.
func (a *Queue[T]) Pop() Option[T] {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.count == 0 {
		return None[T]()
	}

	v := a.inner[a.headIndex]
	a.headIndex = (a.headIndex + 1) % len(a.inner)
	a.count -= 1

	return Some(v)
}

// Peek returns the element at the front of the queue without removing it.
// Returns Some(value) if the queue contains elements, or None() if empty.
// This operation uses a read lock, allowing multiple concurrent Peek operations.
func (a *Queue[T]) Peek() Option[T] {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.count == 0 {
		return None[T]()
	}

	return Some(a.inner[a.headIndex])
}

// Count returns the current number of elements in the queue.
func (a *Queue[T]) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.count
}

// Next returns the next element from the queue, blocking if the queue is empty.
// If the queue contains elements, it immediately returns the first element (same as Pop().Raw()).
// If the queue is empty, it blocks until another goroutine pushes an element via Push().
//
// This method is useful for consumer patterns where you want to continuously process
// items as they become available without polling. Unlike Pop(), which returns None()
// for empty queues, Next() guarantees to return a value by waiting for one.
//
// Warning: This method can block indefinitely if no items are pushed to the queue.
// Use with caution in contexts where blocking is not acceptable.
func (a *Queue[T]) Next() T {
	item, ok := a.Pop().Get()
	if ok {
		return item
	}

	return <-a.nextChan
}
