package kingconc_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BigTony299/kingconc"
	"github.com/stretchr/testify/assert"
)

func TestWorkerPool_CommonBehavior(t *testing.T) {
	testCases := []struct {
		name        string
		poolFactory func() kingconc.WorkerPool
		poolSize    int
	}{
		{
			name:        "Static-1",
			poolFactory: func() kingconc.WorkerPool { return kingconc.NewWorkerPoolStatic(1) },
			poolSize:    1,
		},
		{
			name:        "Static-8",
			poolFactory: func() kingconc.WorkerPool { return kingconc.NewWorkerPoolStatic(8) },
			poolSize:    8,
		},
		{
			name:        "Dynamic",
			poolFactory: func() kingconc.WorkerPool { return kingconc.NewWorkerPoolDynamic() },
			poolSize:    -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("SingleTask", func(t *testing.T) {
				runSingleTaskTest(t, tc.poolFactory())
			})

			t.Run("MultipleTasks", func(t *testing.T) {
				runMultipleTasksTest(t, tc.poolFactory())
			})

			t.Run("ConcurrentTasks", func(t *testing.T) {
				runConcurrentTasksTest(t, tc.poolFactory())
			})

			t.Run("TaskCompletion", func(t *testing.T) {
				runTaskCompletionTest(t, tc.poolFactory())
			})

			t.Run("LongRunningTasks", func(t *testing.T) {
				runLongRunningTasksTest(t, tc.poolFactory())
			})
		})
	}
}

func runSingleTaskTest(t *testing.T, pool kingconc.WorkerPool) {
	assert := assert.New(t)

	var executed int64
	done := make(chan struct{}, 1)

	task := func() {
		atomic.StoreInt64(&executed, 1)
		done <- struct{}{}
	}

	pool.Go(task)

	select {
	case <-done:
		assert.Equal(int64(1), atomic.LoadInt64(&executed), "task should execute exactly once")
	case <-time.After(time.Second):
		t.Fatal("task did not complete within timeout")
	}
}

func runMultipleTasksTest(t *testing.T, pool kingconc.WorkerPool) {
	assert := assert.New(t)

	const numTasks = 10
	var counter int64
	var wg sync.WaitGroup

	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		pool.Go(createCounterTask(&counter, &wg))
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		assert.Equal(int64(numTasks), atomic.LoadInt64(&counter), "all tasks should execute")
	case <-time.After(2 * time.Second):
		t.Fatal("tasks did not complete within timeout")
	}
}

func runConcurrentTasksTest(t *testing.T, pool kingconc.WorkerPool) {
	assert := assert.New(t)

	const numTasks = 100
	var counter int64
	var wg sync.WaitGroup

	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		pool.Go(createCounterTask(&counter, &wg))
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		assert.Equal(
			int64(numTasks),
			atomic.LoadInt64(&counter),
			"all concurrent tasks should execute",
		)
	case <-time.After(3 * time.Second):
		t.Fatalf("concurrent tasks did not complete within timeout, completed: %d/%d",
			atomic.LoadInt64(&counter), numTasks)
	}
}

func runTaskCompletionTest(t *testing.T, pool kingconc.WorkerPool) {
	assert := assert.New(t)

	const numTasks = 20
	results := make(chan int, numTasks)

	for i := 0; i < numTasks; i++ {
		taskID := i
		pool.Go(func() {
			time.Sleep(time.Millisecond)
			results <- taskID
		})
	}

	received := make(map[int]bool)
	for i := 0; i < numTasks; i++ {
		select {
		case taskID := <-results:
			received[taskID] = true
		case <-time.After(2 * time.Second):
			t.Fatal("did not receive all task results within timeout")
		}
	}

	assert.Equal(numTasks, len(received), "should receive results from all tasks")
}

func runLongRunningTasksTest(t *testing.T, pool kingconc.WorkerPool) {
	assert := assert.New(t)

	const numTasks = 5
	const taskDuration = 100 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(numTasks)

	start := time.Now()

	for i := 0; i < numTasks; i++ {
		pool.Go(createDelayTask(taskDuration, &wg))
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		duration := time.Since(start)
		t.Logf("Long running tasks completed in %v", duration)

		sequentialTime := time.Duration(numTasks) * taskDuration
		if duration >= sequentialTime {
			t.Logf("Tasks appear to run sequentially (expected for single-worker pools)")
		} else {
			t.Logf("Tasks appear to run concurrently (expected for multi-worker pools)")
		}

		maxTime := sequentialTime + (200 * time.Millisecond)
		assert.Less(duration, maxTime, "tasks should complete within reasonable time")
	case <-time.After(5 * time.Second):
		t.Fatal("long running tasks did not complete within timeout")
	}
}

func TestWorkerPool_PerformanceComparison(t *testing.T) {
	const numTasks = 50000
	const workDuration = time.Microsecond * 25

	scenarios := []struct {
		name     string
		pool     kingconc.WorkerPool
		executor func(pool kingconc.WorkerPool, tasks []kingconc.Task)
	}{
		{
			name: "Raw Goroutines",
			pool: nil,
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					go func(t kingconc.Task) {
						defer wg.Done()
						t()
					}(task)
				}
				wg.Wait()
			},
		},
		{
			name: "WorkerPoolStatic-1",
			pool: kingconc.NewWorkerPoolStatic(1),
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					taskCopy := task
					pool.Go(func() {
						defer wg.Done()
						taskCopy()
					})
				}
				wg.Wait()
			},
		},
		{
			name: "WorkerPoolSync",
			pool: kingconc.NewWorkerPoolSync(),
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					taskCopy := task
					pool.Go(func() {
						defer wg.Done()
						taskCopy()
					})
				}
				wg.Wait()
			},
		},
		{
			name: "WorkerPoolStatic-10",
			pool: kingconc.NewWorkerPoolStatic(10),
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					taskCopy := task
					pool.Go(func() {
						defer wg.Done()
						taskCopy()
					})
				}
				wg.Wait()
			},
		},
		{
			name: "WorkerPoolStatic-100",
			pool: kingconc.NewWorkerPoolStatic(100),
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					taskCopy := task
					pool.Go(func() {
						defer wg.Done()
						taskCopy()
					})
				}
				wg.Wait()
			},
		},
		{
			name: "WorkerPoolStatic-1000",
			pool: kingconc.NewWorkerPoolStatic(1000),
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					taskCopy := task
					pool.Go(func() {
						defer wg.Done()
						taskCopy()
					})
				}
				wg.Wait()
			},
		},
		{
			name: "WorkerPoolDynamic",
			pool: kingconc.NewWorkerPoolDynamic(),
			executor: func(pool kingconc.WorkerPool, tasks []kingconc.Task) {
				var wg sync.WaitGroup
				wg.Add(len(tasks))

				for _, task := range tasks {
					taskCopy := task
					pool.Go(func() {
						defer wg.Done()
						taskCopy()
					})
				}
				wg.Wait()
			},
		},
	}

	tasks := make([]kingconc.Task, numTasks)
	for i := range tasks {
		tasks[i] = createDelayTask(workDuration, nil)
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Allow some time for any previous goroutines to clean up
			time.Sleep(10 * time.Millisecond)

			// Force garbage collection before measurement
			runtime.GC()
			runtime.GC() // Run twice to ensure cleanup

			start := time.Now()
			scenario.executor(scenario.pool, tasks)
			duration := time.Since(start)

			tasksPerSecond := float64(numTasks) / duration.Seconds()
			t.Logf("%s: completed %d tasks in %v (%.0f tasks/sec)",
				scenario.name, numTasks, duration, tasksPerSecond)

			// Another small cleanup pause after intensive operations
			runtime.GC()
		})
	}
}

func TestWorkerPoolSync_Behavior(t *testing.T) {
	assert := assert.New(t)
	pool := kingconc.NewWorkerPoolSync()

	t.Run("SingleTask", func(t *testing.T) {
		var executed int64
		pool.Go(func() {
			atomic.StoreInt64(&executed, 1)
		})
		assert.Equal(int64(1), atomic.LoadInt64(&executed), "task should execute immediately")
	})

	t.Run("MultipleTasks", func(t *testing.T) {
		var counter int64
		const numTasks = 10

		for i := 0; i < numTasks; i++ {
			pool.Go(func() {
				atomic.AddInt64(&counter, 1)
			})
		}
		assert.Equal(int64(numTasks), atomic.LoadInt64(&counter), "all tasks should execute immediately")
	})

	t.Run("TasksExecuteInCallingGoroutine", func(t *testing.T) {
		var executed bool
		pool.Go(func() {
			executed = true
		})
		assert.True(executed, "task should execute synchronously before Go() returns")
	})
}

func TestWorkerPoolStatic_ZeroSize(t *testing.T) {
	pool := kingconc.NewWorkerPoolStatic(0)

	var executed int64
	done := make(chan struct{})

	pool.Go(func() {
		atomic.StoreInt64(&executed, 1)
		done <- struct{}{}
	})

	select {
	case <-done:
		t.Log("Zero-size pool successfully executed task")
	case <-time.After(100 * time.Millisecond):
		t.Log("Zero-size pool did not execute task (expected behavior)")
	}
}

func createCounterTask(counter *int64, wg *sync.WaitGroup) kingconc.Task {
	return func() {
		if wg != nil {
			defer wg.Done()
		}
		atomic.AddInt64(counter, 1)
	}
}

func createDelayTask(duration time.Duration, wg *sync.WaitGroup) kingconc.Task {
	return func() {
		if wg != nil {
			defer wg.Done()
		}
		time.Sleep(duration)
	}
}

func createPanicTask() kingconc.Task {
	return func() {
		panic("test panic")
	}
}

func createChannelSignalTask(ch chan<- struct{}) kingconc.Task {
	return func() {
		ch <- struct{}{}
	}
}
