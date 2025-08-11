package kingconc_test

import (
	"sync"
	"testing"
	"time"

	"github.com/BigTony299/kingconc"
	"github.com/stretchr/testify/assert"
)

func TestNewAtomic(t *testing.T) {
	assert := assert.New(t)

	t.Run("integer value", func(t *testing.T) {
		const value = 42
		atomic := kingconc.NewAtomic(value)

		result := atomic.Take()
		assert.True(result.Some(), "NewAtomic should create atomic with Some value")
		assert.Equal(value, result.Raw(), "should contain the initialized value")

		// Should be empty after take
		result2 := atomic.Take()
		assert.True(result2.None(), "should be empty after take")
	})

	t.Run("string value", func(t *testing.T) {
		const value = "hello"
		atomic := kingconc.NewAtomic(value)

		result := atomic.Take()
		assert.True(result.Some(), "should contain string value")
		assert.Equal(value, result.Raw(), "should contain the initialized string")
	})

	t.Run("struct value", func(t *testing.T) {
		type TestStruct struct {
			Name string
			ID   int
		}
		value := TestStruct{Name: "test", ID: 123}
		atomic := kingconc.NewAtomic(value)

		result := atomic.Take()
		assert.True(result.Some(), "should contain struct value")
		assert.Equal(value, result.Raw(), "should contain the initialized struct")
	})
}

func TestAtomic_ZeroValue(t *testing.T) {
	assert := assert.New(t)

	var atomic kingconc.Atomic[string]

	result := atomic.Take()
	assert.True(result.None(), "zero value atomic should be empty")
}

func TestAtomic_Put(t *testing.T) {
	assert := assert.New(t)

	t.Run("single value", func(t *testing.T) {
		const value = 42
		atomic := kingconc.NewAtomic(0) // Start with zero, then put value

		atomic.Put(value)

		result := atomic.Take()
		assert.True(result.Some(), "should contain value after Put")
		assert.Equal(value, result.Raw(), "should contain the put value")
	})

	t.Run("replace value", func(t *testing.T) {
		const firstValue = "first"
		const secondValue = "second"
		atomic := kingconc.NewAtomic(firstValue)

		atomic.Put(secondValue)

		result := atomic.Take()
		assert.True(result.Some(), "should contain value after Put")
		assert.Equal(secondValue, result.Raw(), "should contain the latest value")
	})
}

func TestAtomic_Take(t *testing.T) {
	assert := assert.New(t)

	t.Run("empty atomic", func(t *testing.T) {
		var atomic kingconc.Atomic[int]

		result := atomic.Take()

		assert.True(result.None(), "take from empty atomic should return None")
	})

	t.Run("single value", func(t *testing.T) {
		const value = "test"
		atomic := kingconc.NewAtomic(value)

		result := atomic.Take()

		assert.True(result.Some(), "should return Some")
		assert.Equal(value, result.Raw(), "should return initialized value")

		// Second take should return None
		result2 := atomic.Take()
		assert.True(result2.None(), "second take should return None")
	})
}

func TestAtomic_Update(t *testing.T) {
	assert := assert.New(t)

	t.Run("empty atomic", func(t *testing.T) {
		var atomic kingconc.Atomic[int]
		called := false

		atomic.Update(func(v int) int {
			called = true
			return v + 1
		})

		assert.False(called, "function should not be called on empty atomic")
		result := atomic.Take()
		assert.True(result.None(), "atomic should remain empty")
	})

	t.Run("increment value", func(t *testing.T) {
		const initialValue = 5
		atomic := kingconc.NewAtomic(initialValue)

		atomic.Update(func(v int) int {
			return v + 10
		})

		result := atomic.Take()
		assert.True(result.Some(), "should contain updated value")
		assert.Equal(15, result.Raw(), "should contain incremented value")
	})

	t.Run("string transformation", func(t *testing.T) {
		const initialValue = "hello"
		atomic := kingconc.NewAtomic(initialValue)

		atomic.Update(func(v string) string {
			return v + " world"
		})

		result := atomic.Take()
		assert.Equal("hello world", result.Raw(), "should contain transformed string")
	})
}

func TestAtomic_ConcurrentOperations(t *testing.T) {
	assert := assert.New(t)

	t.Run("mixed concurrent operations", func(t *testing.T) {
		var atomic kingconc.Atomic[int]
		var wg sync.WaitGroup
		const duration = 100 * time.Millisecond

		// Concurrent putter
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			counter := 0
			for time.Since(start) < duration {
				atomic.Put(counter)
				counter++
				time.Sleep(time.Microsecond)
			}
		}()

		// Concurrent taker
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			for time.Since(start) < duration {
				atomic.Take()
				time.Sleep(time.Microsecond * 2)
			}
		}()

		// Concurrent updater
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			for time.Since(start) < duration {
				atomic.Update(func(v int) int { return v + 1000 })
				time.Sleep(time.Microsecond * 3)
			}
		}()

		wg.Wait()

		// Test should complete without panics or deadlocks
		assert.True(true, "concurrent operations completed successfully")
	})
}

func TestAtomic_Compare(t *testing.T) {
	assert := assert.New(t)

	t.Run("empty atomic returns false", func(t *testing.T) {
		var atomic kingconc.Atomic[int]
		called := false

		result := atomic.Compare(func(v int) bool {
			called = true
			return v > 0
		})

		assert.False(result, "Compare on empty atomic should return false")
		assert.False(called, "predicate function should not be called on empty atomic")
	})

	t.Run("some value returns false when predicate fails", func(t *testing.T) {
		atomic := kingconc.NewAtomic(5)
		called := false

		result := atomic.Compare(func(v int) bool {
			called = true
			return v > 10 // 5 is not greater than 10
		})

		assert.False(result, "Compare should return false when predicate fails")
		assert.True(called, "predicate function should be called")
	})

	t.Run("some value returns true when predicate succeeds", func(t *testing.T) {
		atomic := kingconc.NewAtomic(15)
		called := false

		result := atomic.Compare(func(v int) bool {
			called = true
			return v > 10 // 15 is greater than 10
		})

		assert.True(result, "Compare should return true when predicate succeeds")
		assert.True(called, "predicate function should be called")
	})
}

func TestNewAtomicNumber(t *testing.T) {
	assert := assert.New(t)

	t.Run("integer value", func(t *testing.T) {
		const value = 42
		atomic := kingconc.NewAtomicNumber(value)

		result := atomic.Take()
		assert.True(result.Some(), "NewAtomicNumber should create atomic with Some value")
		assert.Equal(value, result.Raw(), "should contain the initialized value")
	})

	t.Run("float value", func(t *testing.T) {
		const value = 3.14
		atomic := kingconc.NewAtomicNumber(value)

		result := atomic.Take()
		assert.True(result.Some(), "should contain float value")
		assert.Equal(value, result.Raw(), "should contain the initialized float")
	})

	t.Run("unsigned integer", func(t *testing.T) {
		const value uint32 = 100
		atomic := kingconc.NewAtomicNumber(value)

		result := atomic.Take()
		assert.True(result.Some(), "should contain uint value")
		assert.Equal(value, result.Raw(), "should contain the initialized uint")
	})
}

func TestAtomicNumber_Comparison(t *testing.T) {
	assert := assert.New(t)

	t.Run("Gte - greater than or equal", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		assert.True(atomic.Gte(10), "10 >= 10 should be true")
		assert.True(atomic.Gte(5), "10 >= 5 should be true")
		assert.False(atomic.Gte(15), "10 >= 15 should be false")
	})

	t.Run("Gt - greater than", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		assert.False(atomic.Gt(10), "10 > 10 should be false")
		assert.True(atomic.Gt(5), "10 > 5 should be true")
		assert.False(atomic.Gt(15), "10 > 15 should be false")
	})

	t.Run("Lt - less than", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		assert.False(atomic.Lt(10), "10 < 10 should be false")
		assert.False(atomic.Lt(5), "10 < 5 should be false")
		assert.True(atomic.Lt(15), "10 < 15 should be true")
	})

	t.Run("Lte - less than or equal", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		assert.True(atomic.Lte(10), "10 <= 10 should be true")
		assert.False(atomic.Lte(5), "10 <= 5 should be false")
		assert.True(atomic.Lte(15), "10 <= 15 should be true")
	})

	t.Run("Eq - equal", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		assert.True(atomic.Eq(10), "10 == 10 should be true")
		assert.False(atomic.Eq(5), "10 == 5 should be false")
		assert.False(atomic.Eq(15), "10 == 15 should be false")
	})

	t.Run("Ne - not equal", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		assert.False(atomic.Ne(10), "10 != 10 should be false")
		assert.True(atomic.Ne(5), "10 != 5 should be true")
		assert.True(atomic.Ne(15), "10 != 15 should be true")
	})

	t.Run("empty atomic returns false for all comparisons", func(t *testing.T) {
		var atomic kingconc.AtomicNumber[int]
		assert.False(atomic.Gte(0), "empty atomic Gte should return false")
		assert.False(atomic.Gt(0), "empty atomic Gt should return false")
		assert.False(atomic.Lt(0), "empty atomic Lt should return false")
		assert.False(atomic.Lte(0), "empty atomic Lte should return false")
		assert.False(atomic.Eq(0), "empty atomic Eq should return false")
		assert.False(atomic.Ne(0), "empty atomic Ne should return false")
	})
}

func TestAtomicNumber_Increment(t *testing.T) {
	assert := assert.New(t)

	t.Run("increment existing value", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(5)
		result := atomic.Increment()
		assert.Equal(6, result, "increment should return new value")

		// Verify the value is stored
		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(6, stored.Raw())
	})

	t.Run("increment empty atomic", func(t *testing.T) {
		var atomic kingconc.AtomicNumber[int]
		result := atomic.Increment()
		assert.Equal(1, result, "increment on empty should return 1")

		// Verify the value is stored
		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(1, stored.Raw())
	})

	t.Run("concurrent increments", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		const numGoroutines = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for range numGoroutines {
			go func() {
				defer wg.Done()
				atomic.Increment()
			}()
		}

		wg.Wait()
		result := atomic.Take()
		assert.True(result.Some())
		assert.Equal(numGoroutines, result.Raw(), "concurrent increments should be atomic")
	})
}

func TestAtomicNumber_Decrement(t *testing.T) {
	assert := assert.New(t)

	t.Run("decrement existing value", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(5)
		result := atomic.Decrement()
		assert.Equal(4, result, "decrement should return new value")

		// Verify the value is stored
		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(4, stored.Raw())
	})

	t.Run("decrement empty atomic", func(t *testing.T) {
		var atomic kingconc.AtomicNumber[int]
		result := atomic.Decrement()
		assert.Equal(-1, result, "decrement on empty should return -1")

		// Verify the value is stored
		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(-1, stored.Raw())
	})

	t.Run("decrement unsigned from empty wraps", func(t *testing.T) {
		var atomic kingconc.AtomicNumber[uint8]
		result := atomic.Decrement()
		assert.Equal(uint8(255), result, "decrement uint8 from 0 should wrap to 255")
	})
}

func TestAtomicNumber_Add(t *testing.T) {
	assert := assert.New(t)

	t.Run("add to existing value", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		result := atomic.Add(5)
		assert.Equal(15, result, "add should return new value")

		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(15, stored.Raw())
	})

	t.Run("add to empty atomic", func(t *testing.T) {
		var atomic kingconc.AtomicNumber[int]
		result := atomic.Add(7)
		assert.Equal(7, result, "add to empty should return delta")

		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(7, stored.Raw())
	})

	t.Run("add negative value", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		result := atomic.Add(-3)
		assert.Equal(7, result, "add negative should subtract")
	})
}

func TestAtomicNumber_Sub(t *testing.T) {
	assert := assert.New(t)

	t.Run("subtract from existing value", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		result := atomic.Sub(3)
		assert.Equal(7, result, "sub should return new value")

		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(7, stored.Raw())
	})

	t.Run("subtract from empty atomic", func(t *testing.T) {
		var atomic kingconc.AtomicNumber[int]
		result := atomic.Sub(5)
		assert.Equal(-5, result, "sub from empty should return -delta")

		stored := atomic.Take()
		assert.True(stored.Some())
		assert.Equal(-5, stored.Raw())
	})

	t.Run("subtract negative value", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(10)
		result := atomic.Sub(-3)
		assert.Equal(13, result, "sub negative should add")
	})
}

func TestAtomicNumber_OnceCallback(t *testing.T) {
	assert := assert.New(t)

	t.Run("once callback triggered when condition met", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		called := false

		atomic.Once(func(v int) bool { return v >= 5 }, func() { called = true })

		atomic.Increment()
		assert.False(called, "callback should not trigger at 1")

		atomic.Add(4)
		assert.True(called, "callback should trigger when reaching 5")
	})

	t.Run("once callback only executes once", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		count := 0

		atomic.Once(func(v int) bool { return v >= 3 }, func() { count++ })

		atomic.Add(5)
		atomic.Add(2)
		atomic.Add(3)

		assert.Equal(1, count, "callback should execute only once")
	})

	t.Run("multiple once callbacks", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		called1 := false
		called2 := false

		atomic.Once(func(v int) bool { return v >= 3 }, func() { called1 = true })
		atomic.Once(func(v int) bool { return v >= 7 }, func() { called2 = true })

		atomic.Add(5)
		assert.True(called1, "first callback should trigger")
		assert.False(called2, "second callback should not trigger yet")

		atomic.Add(3)
		assert.True(called2, "second callback should now trigger")
	})

	t.Run("nil functions ignored", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)

		atomic.Once(nil, func() {})
		atomic.Once(func(v int) bool { return true }, nil)

		atomic.Increment()
		assert.True(true, "should not panic with nil functions")
	})
}

func TestAtomicNumber_EveryCallback(t *testing.T) {
	assert := assert.New(t)

	t.Run("every callback triggered multiple times", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		count := 0

		atomic.Every(func(v int) bool { return v%2 == 0 }, func() { count++ })

		atomic.Add(2)
		assert.Equal(1, count, "callback should trigger for even value 2")

		atomic.Add(2)
		assert.Equal(2, count, "callback should trigger again for even value 4")

		atomic.Add(1)
		assert.Equal(2, count, "callback should not trigger for odd value 5")

		atomic.Add(1)
		assert.Equal(3, count, "callback should trigger for even value 6")
	})

	t.Run("every callback with multiple conditions", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		evenCount := 0
		highCount := 0

		atomic.Every(func(v int) bool { return v%2 == 0 }, func() { evenCount++ })
		atomic.Every(func(v int) bool { return v > 10 }, func() { highCount++ })

		atomic.Add(8)
		assert.Equal(1, evenCount, "even callback should trigger")
		assert.Equal(0, highCount, "high callback should not trigger")

		atomic.Add(4)
		assert.Equal(2, evenCount, "even callback should trigger again")
		assert.Equal(1, highCount, "high callback should trigger for value 12")
	})
}

func TestAtomicNumber_CallbackIntegration(t *testing.T) {
	assert := assert.New(t)

	t.Run("callbacks work with all numeric operations", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		events := []string{}

		atomic.Once(func(v int) bool { return v == 1 }, func() { events = append(events, "increment") })
		atomic.Once(func(v int) bool { return v == 6 }, func() { events = append(events, "add") })
		atomic.Once(func(v int) bool { return v == 3 }, func() { events = append(events, "sub") })
		atomic.Once(func(v int) bool { return v == 2 }, func() { events = append(events, "decrement") })

		atomic.Increment()
		atomic.Add(5)
		atomic.Sub(3)
		atomic.Decrement()

		expectedEvents := []string{"increment", "add", "sub", "decrement"}
		assert.Equal(expectedEvents, events, "all callbacks should trigger in order")
	})

	t.Run("concurrent operations with callbacks", func(t *testing.T) {
		atomic := kingconc.NewAtomicNumber(0)
		callbackCount := 0
		var mu sync.Mutex

		atomic.Every(func(v int) bool { return v%10 == 0 && v > 0 }, func() {
			mu.Lock()
			callbackCount++
			mu.Unlock()
		})

		var wg sync.WaitGroup
		const numGoroutines = 50

		wg.Add(numGoroutines)
		for range numGoroutines {
			go func() {
				defer wg.Done()
				atomic.Add(2)
			}()
		}

		wg.Wait()

		mu.Lock()
		finalCount := callbackCount
		mu.Unlock()

		finalValue := atomic.Take()
		assert.Equal(100, finalValue.Raw(), "final value should be 100")
		assert.Equal(10, finalCount, "callback should trigger 10 times for multiples of 10")
	})
}
