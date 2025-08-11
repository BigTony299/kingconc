package kingconc_test

import (
	"sync"
	"testing"
	"time"

	"github.com/BigTony299/kingconc"
	"github.com/stretchr/testify/assert"
)

func TestNewBroadcaster(t *testing.T) {
	assert := assert.New(t)

	t.Run("given nil options then use defaults", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		assert.NotNil(broadcaster)
	})

	t.Run("given custom pool then use it", func(t *testing.T) {
		t.Parallel()

		pool := kingconc.NewWorkerPoolStatic(1)
		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			WorkerPool: kingconc.Some[kingconc.WorkerPool](pool),
		})
		assert.NotNil(broadcaster)
	})

	t.Run("given buffer size then create buffered broadcaster", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			BufferSize: kingconc.Some(10),
		})
		assert.NotNil(broadcaster)
	})

	t.Run("given custom pool and buffer size then use both", func(t *testing.T) {
		t.Parallel()

		pool := kingconc.NewWorkerPoolStatic(1)
		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			WorkerPool: kingconc.Some[kingconc.WorkerPool](pool),
			BufferSize: kingconc.Some(15),
		})
		assert.NotNil(broadcaster)
	})
}

func TestBroadcaster_Subscribe(t *testing.T) {
	assert := assert.New(t)

	t.Run("given active broadcaster then return channel", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		ch := broadcaster.Subscribe()
		assert.NotNil(ch)
	})

	t.Run("given closed broadcaster then return nil", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		broadcaster.Close()
		ch := broadcaster.Subscribe()
		assert.Nil(ch)
	})

	t.Run("given multiple subscribers then all receive channels", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		ch1 := broadcaster.Subscribe()
		ch2 := broadcaster.Subscribe()
		ch3 := broadcaster.Subscribe()

		assert.NotNil(ch1)
		assert.NotNil(ch2)
		assert.NotNil(ch3)
		assert.NotEqual(ch1, ch2)
		assert.NotEqual(ch2, ch3)
	})
}

func TestBroadcaster_Broadcast(t *testing.T) {
	assert := assert.New(t)

	t.Run("given single subscriber then receive message", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			WorkerPool: kingconc.Some[kingconc.WorkerPool](kingconc.NewWorkerPoolDynamic()),
		})
		ch := broadcaster.Subscribe()

		go broadcaster.Broadcast("test message")

		select {
		case msg := <-ch:
			assert.Equal("test message", msg)
		case <-time.After(100 * time.Millisecond):
			assert.FailNow("timeout waiting for message")
		}
	})

	t.Run("given multiple subscribers then all receive message", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			WorkerPool: kingconc.Some[kingconc.WorkerPool](kingconc.NewWorkerPoolDynamic()),
		})
		ch1 := broadcaster.Subscribe()
		ch2 := broadcaster.Subscribe()
		ch3 := broadcaster.Subscribe()

		var wg sync.WaitGroup
		wg.Add(3)

		checkMessage := func(ch <-chan string, expected string) {
			defer wg.Done()
			select {
			case msg := <-ch:
				assert.Equal(expected, msg)
			case <-time.After(100 * time.Millisecond):
				assert.FailNow("timeout waiting for message")
			}
		}

		go checkMessage(ch1, "broadcast message")
		go checkMessage(ch2, "broadcast message")
		go checkMessage(ch3, "broadcast message")

		time.Sleep(10 * time.Millisecond) // Let goroutines start
		broadcaster.Broadcast("broadcast message")

		wg.Wait()
	})

	t.Run("given closed broadcaster then broadcast is ignored", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		ch := broadcaster.Subscribe()

		// Read the close signal first
		broadcaster.Close()
		_, ok := <-ch
		assert.False(ok, "channel should be closed")

		// Now try to broadcast - should be safe but ignored
		broadcaster.Broadcast("ignored message")
	})
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	assert := assert.New(t)

	t.Run("given existing subscriber then remove and close channel", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		ch := broadcaster.Subscribe()

		success := broadcaster.Unsubscribe(ch)
		assert.True(success)

		// Channel should be closed
		_, ok := <-ch
		assert.False(ok, "channel should be closed")
	})

	t.Run("given non-existent subscriber then return false", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		fakeChannel := make(<-chan string)

		success := broadcaster.Unsubscribe(fakeChannel)
		assert.False(success)
	})

	t.Run("given multiple subscribers then unsubscribe only target", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			WorkerPool: kingconc.Some[kingconc.WorkerPool](kingconc.NewWorkerPoolDynamic()),
		})
		ch1 := broadcaster.Subscribe()
		ch2 := broadcaster.Subscribe()
		ch3 := broadcaster.Subscribe()

		success := broadcaster.Unsubscribe(ch2)
		assert.True(success)

		// ch2 should be closed
		_, ok := <-ch2
		assert.False(ok, "ch2 should be closed")

		// ch1 and ch3 should still work
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			select {
			case msg := <-ch1:
				assert.Equal("test", msg)
			case <-time.After(100 * time.Millisecond):
				assert.FailNow("ch1 should receive message")
			}
		}()

		go func() {
			defer wg.Done()
			select {
			case msg := <-ch3:
				assert.Equal("test", msg)
			case <-time.After(100 * time.Millisecond):
				assert.FailNow("ch3 should receive message")
			}
		}()

		time.Sleep(10 * time.Millisecond) // Let goroutines start
		broadcaster.Broadcast("test")
		wg.Wait()
	})
}

func TestBroadcaster_Close(t *testing.T) {
	assert := assert.New(t)

	t.Run("given active subscribers then close all channels", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		ch1 := broadcaster.Subscribe()
		ch2 := broadcaster.Subscribe()
		ch3 := broadcaster.Subscribe()

		broadcaster.Close()

		// All channels should be closed
		_, ok1 := <-ch1
		_, ok2 := <-ch2
		_, ok3 := <-ch3

		assert.False(ok1, "ch1 should be closed")
		assert.False(ok2, "ch2 should be closed")
		assert.False(ok3, "ch3 should be closed")
	})

	t.Run("given already closed broadcaster then multiple close calls are safe", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		ch := broadcaster.Subscribe()

		broadcaster.Close()
		broadcaster.Close() // Should not panic

		_, ok := <-ch
		assert.False(ok, "channel should be closed")
	})

	t.Run("given closed broadcaster then new subscriptions return nil", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{})
		broadcaster.Close()

		ch := broadcaster.Subscribe()
		assert.Nil(ch)
	})
}

func TestBroadcaster_BufferedChannels(t *testing.T) {
	assert := assert.New(t)

	t.Run("given buffered broadcaster then channels can queue messages", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[string](kingconc.BroadcasterOptions{
			BufferSize: kingconc.Some(3),
		})
		ch := broadcaster.Subscribe()

		// Send multiple messages without reading
		broadcaster.Broadcast("msg1")
		broadcaster.Broadcast("msg2")
		broadcaster.Broadcast("msg3")

		// Wait a bit for all messages to be sent
		time.Sleep(10 * time.Millisecond)

		// Should be able to read all messages (order not guaranteed due to TaskPool concurrency)
		receivedMessages := make(map[string]bool)
		receivedMessages[<-ch] = true
		receivedMessages[<-ch] = true
		receivedMessages[<-ch] = true

		// Verify all expected messages were received
		assert.True(receivedMessages["msg1"], "msg1 should be received")
		assert.True(receivedMessages["msg2"], "msg2 should be received")
		assert.True(receivedMessages["msg3"], "msg3 should be received")
		assert.Len(receivedMessages, 3, "should receive exactly 3 unique messages")
	})
}

func TestBroadcaster_ConcurrentAccess(t *testing.T) {
	assert := assert.New(t)

	t.Run("concurrent subscribe/unsubscribe/broadcast operations", func(t *testing.T) {
		t.Parallel()

		broadcaster := kingconc.NewBroadcaster[int](kingconc.BroadcasterOptions{
			WorkerPool: kingconc.Some[kingconc.WorkerPool](kingconc.NewWorkerPoolDynamic()),
		})
		const numWorkers = 10
		const numOperations = 100

		var wg sync.WaitGroup
		wg.Add(numWorkers * 3) // subscribers, broadcasters, unsubscribers

		// Concurrent subscribers
		subscribers := make([]<-chan int, numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func(idx int) {
				defer wg.Done()
				subscribers[idx] = broadcaster.Subscribe()
			}(i)
		}

		// Concurrent broadcasters
		for i := 0; i < numWorkers; i++ {
			go func(value int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					broadcaster.Broadcast(value*numOperations + j)
				}
			}(i)
		}

		// Concurrent unsubscribers (after some delay)
		for i := 0; i < numWorkers; i++ {
			go func(idx int) {
				defer wg.Done()
				time.Sleep(50 * time.Millisecond) // Let some broadcasts happen
				if subscribers[idx] != nil {
					broadcaster.Unsubscribe(subscribers[idx])
				}
			}(i)
		}

		wg.Wait()
		broadcaster.Close()

		// Test should complete without data races or panics
		assert.True(true, "concurrent operations completed successfully")
	})
}
