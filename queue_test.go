package kingconc_test

import (
	"sync"
	"testing"
	"time"

	"github.com/BigTony299/kingconc"
	"github.com/stretchr/testify/assert"
)

func TestNewQueue(t *testing.T) {
	assert := assert.New(t)

	queue := kingconc.NewQueue[string]()

	assert.Equal(0, queue.Count(), "initial count should be 0")

	pop := queue.Pop()
	assert.True(pop.None(), "pop from empty queue should return None")

	peek := queue.Peek()
	assert.True(peek.None(), "peek from empty queue should return None")
}

func TestQueue_Push(t *testing.T) {
	assert := assert.New(t)

	t.Run("single element", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()

		queue.Push(42)

		assert.Equal(1, queue.Count(), "count should be 1 after push")
	})

	t.Run("multiple elements", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()

		queue.Push("first")
		queue.Push("second")
		queue.Push("third")

		assert.Equal(3, queue.Count(), "count should be 3 after three pushes")
	})

	t.Run("different types", func(t *testing.T) {
		type TestStruct struct {
			Name string
			ID   int
		}

		queue := kingconc.NewQueue[TestStruct]()
		testVal := TestStruct{Name: "test", ID: 123}

		queue.Push(testVal)

		assert.Equal(1, queue.Count(), "count should be 1")
		peeked := queue.Peek()
		assert.True(peeked.Some(), "should contain value")
		assert.Equal(testVal, peeked.Raw(), "should contain pushed struct")
	})
}

func TestQueue_Pop(t *testing.T) {
	assert := assert.New(t)

	t.Run("empty queue", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()

		result := queue.Pop()

		assert.True(result.None(), "pop from empty queue should return None")
	})

	t.Run("single element", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()
		const value = "test"

		queue.Push(value)
		result := queue.Pop()

		assert.True(result.Some(), "should return Some")
		assert.Equal(value, result.Raw(), "should return pushed value")
		assert.Equal(0, queue.Count(), "queue should be empty after pop")

		// Second pop should return None
		result2 := queue.Pop()
		assert.True(result2.None(), "second pop should return None")
	})

	t.Run("multiple elements", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()

		queue.Push(1)
		queue.Push(2)
		queue.Push(3)

		first := queue.Pop()
		second := queue.Pop()
		third := queue.Pop()

		assert.Equal(1, first.Raw(), "first pop should return first pushed")
		assert.Equal(2, second.Raw(), "second pop should return second pushed")
		assert.Equal(3, third.Raw(), "third pop should return third pushed")
	})

	t.Run("mixed operations", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()

		queue.Push("a")
		first := queue.Pop()
		queue.Push("b")
		queue.Push("c")
		second := queue.Pop()

		assert.Equal("a", first.Raw(), "first pop should return 'a'")
		assert.Equal("b", second.Raw(), "second pop should return 'b'")
		assert.Equal(1, queue.Count(), "one element should remain")
	})
}

func TestQueue_Peek(t *testing.T) {
	assert := assert.New(t)

	t.Run("empty queue", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()

		result := queue.Peek()

		assert.True(result.None(), "peek from empty queue should return None")
	})

	t.Run("single element", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()
		const value = "test"

		queue.Push(value)
		result := queue.Peek()

		assert.True(result.Some(), "should return Some")
		assert.Equal(value, result.Raw(), "should return pushed value")
		assert.Equal(1, queue.Count(), "queue size should remain unchanged")

		// Peek again should return same value
		result2 := queue.Peek()
		assert.Equal(value, result2.Raw(), "second peek should return same value")
	})

	t.Run("after pop", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()

		queue.Push("first")
		queue.Push("second")

		queue.Pop() // Remove "first"
		result := queue.Peek()

		assert.Equal("second", result.Raw(), "peek should return next element after pop")
	})
}

func TestQueue_Count(t *testing.T) {
	assert := assert.New(t)

	t.Run("empty queue", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()

		assert.Equal(0, queue.Count(), "initial count should be 0")
	})

	t.Run("mixed operations", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()

		queue.Push("a")
		assert.Equal(1, queue.Count(), "count should be 1")

		queue.Pop()
		assert.Equal(0, queue.Count(), "count should be 0 after pop")

		queue.Push("b")
		queue.Push("c")
		assert.Equal(2, queue.Count(), "count should be 2 after two more pushes")

		queue.Pop()
		assert.Equal(1, queue.Count(), "count should be 1 after pop")
	})
}

func TestQueue_FIFOOrdering(t *testing.T) {
	assert := assert.New(t)

	queue := kingconc.NewQueue[int]()

	// Push elements
	for i := range 5 {
		queue.Push(i + 1)
	}

	// Pop elements and verify order
	for i := range 5 {
		result := queue.Pop()
		assert.True(result.Some(), "should have element")
		assert.Equal(i+1, result.Raw(), "should maintain FIFO order")
	}

	// Queue should be empty
	result := queue.Pop()
	assert.True(result.None(), "queue should be empty")
}

func TestQueue_ConcurrentOperations(t *testing.T) {
	assert := assert.New(t)

	t.Run("mixed concurrent operations", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()
		var wg sync.WaitGroup

		// Concurrent pushers
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				queue.Push(i)
				time.Sleep(time.Microsecond)
			}
		}()

		// Concurrent poppers
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				queue.Pop()
				time.Sleep(time.Microsecond * 2)
			}
		}()

		// Concurrent peekers
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				queue.Peek()
			}
		}()

		// Concurrent counters
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				queue.Count()
			}
		}()

		wg.Wait()

		// Test should complete without panics or deadlocks
		assert.True(true, "concurrent operations completed successfully")
	})
}

func TestQueue_Next(t *testing.T) {
	assert := assert.New(t)

	t.Run("next with multiple elements", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()

		queue.Push(1)
		queue.Push(2)
		queue.Push(3)

		first := queue.Next()
		second := queue.Next()
		third := queue.Next()

		assert.Equal(1, first, "first Next should return first pushed")
		assert.Equal(2, second, "second Next should return second pushed")
		assert.Equal(3, third, "third Next should return third pushed")
		assert.Equal(0, queue.Count(), "queue should be empty")
	})

	t.Run("next blocks until push", func(t *testing.T) {
		queue := kingconc.NewQueue[string]()
		const value = "delayed"

		var result string
		var wg sync.WaitGroup

		// Start goroutine that calls Next (will block)
		wg.Add(1)
		go func() {
			defer wg.Done()
			result = queue.Next()
		}()

		// Give the goroutine time to start and block
		time.Sleep(10 * time.Millisecond)

		// Push value to unblock Next
		queue.Push(value)

		wg.Wait()

		assert.Equal(value, result, "should return value pushed after Next call")
	})

	t.Run("concurrent next calls", func(t *testing.T) {
		queue := kingconc.NewQueue[int]()
		const numGoroutines = 5

		results := make([]int, numGoroutines)
		var wg sync.WaitGroup

		// Start multiple goroutines calling Next
		for i := range numGoroutines {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				results[index] = queue.Next()
			}(i)
		}

		// Give goroutines time to start and block
		time.Sleep(10 * time.Millisecond)

		// Push values to unblock the Next calls
		for i := range numGoroutines {
			queue.Push(i * 10) // Use distinct values
		}

		wg.Wait()

		// All goroutines should have received a value
		receivedValues := make(map[int]bool)
		for _, result := range results {
			receivedValues[result] = true
		}

		assert.Equal(numGoroutines, len(receivedValues), "all goroutines should receive distinct values")

		// Verify all expected values were received
		for i := range numGoroutines {
			expectedValue := i * 10
			assert.True(receivedValues[expectedValue], "should receive value %d", expectedValue)
		}
	})
}
