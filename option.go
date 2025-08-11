package kingconc

// Some creates an Option that contains a value.
// It represents the presence of a value of type T.
func Some[T any](t T) Option[T] {
	return Option[T]{
		item: t,
		ok:   true,
	}
}

// None creates an empty Option that contains no value.
// It represents the absence of a value of type T.
func None[T any]() Option[T] {
	return Option[T]{}
}

// Option represents an optional value that may or may not be present.
// It is similar to Rust's Option type or other languages' Maybe type.
// An Option is either Some (containing a value) or None (empty).
// The Option type has minimal memory overhead, storing only the value and a boolean flag.
type Option[T any] struct {
	item T
	ok   bool
}

// Some returns true if the Option contains a value, false otherwise.
func (o Option[T]) Some() bool {
	return o.ok
}

// None returns true if the Option is empty (contains no value), false otherwise.
func (o Option[T]) None() bool {
	return !o.ok
}

// Get returns the value and a boolean indicating whether the Option contains a value.
// If the Option is Some, returns (value, true). If None, returns (zero value, false).
func (o Option[T]) Get() (T, bool) {
	return o.item, o.ok
}

// GetOrDefault returns the contained value if the Option is Some,
// otherwise returns the provided default value.
// This is useful for providing fallback values when the Option is None.
func (o Option[T]) GetOrDefault(t T) T {
	if !o.ok {
		return t
	}
	return o.item
}

// Raw returns the raw value stored in the Option without checking if it's present.
// If the Option is None, this returns the zero value of type T.
// Use Get() or check Some()/None() if you need to distinguish between a zero value and None.
func (o Option[T]) Raw() T {
	return o.item
}
