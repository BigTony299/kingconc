package kingconc

import "sync"

// NewAtomic creates a new Atomic container initialized with the provided value.
func NewAtomic[T any](t T) Atomic[T] {
	return Atomic[T]{
		inner: Some(t),
	}
}

// Atomic provides thread-safe atomic operations on an optional value.
// It wraps an Option[T] with a mutex to ensure all operations are atomic.
// The zero value is ready to use and represents an empty atomic container.
//
// Memory overhead: ~24 bytes + size of T when present.
type Atomic[T any] struct {
	mu sync.Mutex

	inner Option[T]

	onces  []atomicCallback[T]
	everys []atomicCallback[T]
}

// Take atomically retrieves and removes the current value from the container.
// Returns the current value wrapped in Some if present, or None if empty.
// After Take is called, the container becomes empty regardless of the previous state.
//
// This operation is thread-safe and atomic - no other goroutine can observe
// an intermediate state where the value is partially taken.
func (a *Atomic[T]) Take() Option[T] {
	a.mu.Lock()
	defer a.mu.Unlock()

	inner := a.inner
	a.inner = None[T]()

	return inner
}

// Put atomically stores a value in the container.
// Any previously stored value is replaced. The container will contain
// the new value wrapped in Some after this operation completes.
//
// This operation is thread-safe and atomic - other goroutines will
// observe either the old value or the new value, never an intermediate state.
func (a *Atomic[T]) Put(t T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.inner = Some(t)

	a.doCallbacks()
}

// Update atomically applies a transformation function to the current value if present.
// If the container is empty (None), the function is not called and no change occurs.
// If the container contains a value, the function is applied to that value and
// the result replaces the original value.
//
// The transformation function f should be pure and not perform blocking operations
// since it runs while holding the internal mutex.
//
// This operation is thread-safe and atomic - other goroutines will observe
// either the old value or the new transformed value, never an intermediate state.
func (a *Atomic[T]) Update(f func(T) T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.inner.None() {
		return
	}

	newVal := f(a.inner.Raw())

	a.inner = Some(newVal)

	a.doCallbacks()
}

// Compare atomically checks if the current value satisfies the given predicate function.
// Returns false if the container is empty (None), regardless of the predicate.
// If the container contains a value, applies the predicate function to that value
// and returns the result.
//
// The predicate function f should be pure and not perform blocking operations
// since it runs while holding the internal mutex.
//
// This operation is thread-safe and atomic - the comparison is performed on a
// consistent snapshot of the container's state.
func (a *Atomic[T]) Compare(f func(T) bool) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.inner.None() {
		return false
	}

	return f(a.inner.Raw())
}

// Once registers a new callback to be executed when the condition is reached, only once.
//
// If either when or then is nil, the function does nothing.
func (a *Atomic[T]) Once(when func(T) bool, then func()) {
	if when == nil || then == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.onces = append(a.onces, atomicCallback[T]{
		when: when,
		then: then,
	})
}

// Every registers a new callback to be executed when the condition is reached, every time.
//
// If either when or then is nil, the function does nothing.
func (a *Atomic[T]) Every(when func(T) bool, then func()) {
	if when == nil || then == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.everys = append(a.everys, atomicCallback[T]{
		when: when,
		then: then,
	})
}

func (a *Atomic[T]) doCallbacks() {
	notExec := make([]atomicCallback[T], 0, len(a.onces))
	for _, once := range a.onces {
		if once.when(a.inner.Raw()) {
			once.then()
		} else {
			notExec = append(notExec, once)
		}
	}
	a.onces = notExec

	for _, every := range a.everys {
		if every.when(a.inner.Raw()) {
			every.then()
		}
	}
}

type atomicCallback[T any] struct {
	when func(T) bool
	then func()
}

// NewAtomicNumber creates a new AtomicNumber container initialized with the provided value.
// It wraps the value in an Atomic container and provides additional numeric operations
// that are thread-safe and atomic.
func NewAtomicNumber[T Number](t T) AtomicNumber[T] {
	return AtomicNumber[T]{
		Atomic: NewAtomic(t),
	}
}

// AtomicNumber defines an atomic number value.
// It provides additional methods to work with numbers beyond the base Atomic functionality.
// All operations are thread-safe and atomic.
type AtomicNumber[T Number] struct {
	Atomic[T]
}

// Gte atomically checks if the current value is greater than or equal to the given value.
// Returns false if the container is empty (None), regardless of the comparison value.
func (a *AtomicNumber[T]) Gte(v T) bool {
	return a.Compare(func(current T) bool {
		return current >= v
	})
}

// Gt atomically checks if the current value is greater than the given value.
// Returns false if the container is empty (None), regardless of the comparison value.
func (a *AtomicNumber[T]) Gt(v T) bool {
	return a.Compare(func(current T) bool {
		return current > v
	})
}

// Lt atomically checks if the current value is less than the given value.
// Returns false if the container is empty (None), regardless of the comparison value.
func (a *AtomicNumber[T]) Lt(v T) bool {
	return a.Compare(func(current T) bool {
		return current < v
	})
}

// Lte atomically checks if the current value is less than or equal to the given value.
// Returns false if the container is empty (None), regardless of the comparison value.
func (a *AtomicNumber[T]) Lte(v T) bool {
	return a.Compare(func(current T) bool {
		return current <= v
	})
}

// Eq atomically checks if the current value is equal to the given value.
// Returns false if the container is empty (None), regardless of the comparison value.
func (a *AtomicNumber[T]) Eq(v T) bool {
	return a.Compare(func(current T) bool {
		return current == v
	})
}

// Ne atomically checks if the current value is not equal to the given value.
// Returns false if the container is empty (None), regardless of the comparison value.
func (a *AtomicNumber[T]) Ne(v T) bool {
	return a.Compare(func(current T) bool {
		return current != v
	})
}

// Increment atomically increments the current value by 1 and returns the new value.
// If the container is empty (None), it initializes with 1 and returns 1.
// This operation is thread-safe and atomic.
func (a *AtomicNumber[T]) Increment() T {
	return a.Add(T(1))
}

// Decrement atomically decrements the current value by 1 and returns the new value.
// If the container is empty (None), it initializes with 0 and then decrements to get the new value.
// For unsigned types, decrementing from 0 will wrap around according to the type's behavior.
// This operation is thread-safe and atomic.
func (a *AtomicNumber[T]) Decrement() T {
	return a.Sub(T(1))
}

// Add atomically adds the given delta to the current value and returns the new value.
// If the container is empty (None), it initializes with delta and returns delta.
// This operation is thread-safe and atomic.
func (a *AtomicNumber[T]) Add(delta T) T {
	if a.inner.None() {
		a.Put(delta)
		return delta
	}

	var result T
	a.Update(func(current T) T {
		result = current + delta
		return result
	})
	return result
}

// Sub atomically subtracts the given delta from the current value and returns the new value.
// If the container is empty (None), it initializes with -delta and returns -delta.
// This operation is thread-safe and atomic.
func (a *AtomicNumber[T]) Sub(delta T) T {
	if a.inner.None() {
		result := -delta
		a.Put(result)
		return result
	}

	var result T
	a.Update(func(current T) T {
		result = current - delta
		return result
	})
	return result
}
