package kingconc

import (
	"sync"
)

// Broadcaster provides thread-safe pub/sub messaging with concurrent delivery.
// Uses a Pool to deliver messages to subscribers without blocking.
type Broadcaster[T any] struct {
	mu sync.RWMutex

	subscribers []chan T
	workerPool  WorkerPool
	bufferSize  int
	closed      bool
}

// NewBroadcaster creates a broadcaster with unbuffered channels.
// Uses provided pool for message delivery, or SyncPool if nil.
func NewBroadcaster[T any](options BroadcasterOptions) *Broadcaster[T] {
	return &Broadcaster[T]{
		workerPool: options.WorkerPool.GetOrDefault(NewWorkerPoolSync()),
		bufferSize: options.BufferSize.GetOrDefault(0),
	}
}

// Subscribe creates and returns a new subscription channel.
// Returns nil if broadcaster is closed.
func (o *Broadcaster[T]) Subscribe() <-chan T {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return nil
	}

	var subscription chan T
	if o.bufferSize > 0 {
		subscription = make(chan T, o.bufferSize)
	} else {
		subscription = make(chan T)
	}

	o.subscribers = append(o.subscribers, subscription)

	return subscription
}

// Broadcast sends a message to all subscribers concurrently.
// Messages are dropped if subscriber channels are full or not ready.
// Ignored if broadcaster is closed.
func (o *Broadcaster[T]) Broadcast(t T) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.closed {
		return
	}

	for _, c := range o.subscribers {
		o.workerPool.Go(func() {
			select {
			case c <- t:
			default:
			}
		})
	}
}

// Unsubscribe removes and closes a subscriber channel.
// Returns true if the channel was found and unsubscribed.
func (o *Broadcaster[T]) Unsubscribe(ch <-chan T) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	for i, sub := range o.subscribers {
		if sub == ch {
			// Remove from slice
			o.subscribers = append(o.subscribers[:i], o.subscribers[i+1:]...)
			// Close the channel
			close(sub)
			return true
		}
	}
	return false
}

// Close shuts down the broadcaster and closes all subscriber channels.
// Subsequent Subscribe() calls return nil, Broadcast() calls are ignored.
// Safe to call multiple times.
func (o *Broadcaster[T]) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return
	}

	o.closed = true

	// Close all subscriber channels
	for _, ch := range o.subscribers {
		close(ch)
	}

	// Clear subscribers slice
	o.subscribers = o.subscribers[:0]
}

type BroadcasterOptions struct {
	WorkerPool Option[WorkerPool]
	BufferSize Option[int]
}
