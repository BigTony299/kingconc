package kingconc

// WorkerPool is a simple interface to implement worker pools.
type WorkerPool interface {
	// Go provides a simple way to pass a task to a goroutine.
	Go(Task)
}

var (
	_ WorkerPool = (*WorkerPoolStatic)(nil)
	_ WorkerPool = (*WorkerPoolDynamic)(nil)
	_ WorkerPool = (*WorkerPoolSync)(nil)
)

// NewWorkerPoolStatic initializes a new WorkerPoolStatic with n goroutines ready.
func NewWorkerPoolStatic(size int) *WorkerPoolStatic {
	if size < 1 {
		size = 1
	}

	m := &WorkerPoolStatic{
		ready: make(chan chan<- Task, size),
	}

	for range size {
		input := make(chan Task)

		go func() {
			m.ready <- input

			for task := range input {
				task()

				m.ready <- input
			}
		}()
	}

	return m
}

// WorkerPoolStatic is a pool that starts n goroutines at initialization.
type WorkerPoolStatic struct {
	ready chan chan<- Task
}

// Go implements WorkerPool.
func (t *WorkerPoolStatic) Go(task Task) {
	(<-t.ready) <- task
}

// NewWorkerPoolDynamic initializes a new WorkerPoolDynamic.
func NewWorkerPoolDynamic() *WorkerPoolDynamic {
	return &WorkerPoolDynamic{
		queue: NewQueue[chan<- Task](),
	}
}

// WorkerPoolDynamic is a pool that scales to the demand.
// Once a goroutine has been created, it will always remain active.
type WorkerPoolDynamic struct {
	queue Queue[chan<- Task]
}

// Go implements WorkerPool.
func (t *WorkerPoolDynamic) Go(task Task) {
	if opt := t.queue.Pop(); opt.Some() {
		opt.Raw() <- task
	} else {
		tasks := make(chan Task)

		go func() {
			for task := range tasks {
				task()

				t.queue.Push(tasks)
			}
		}()

		tasks <- task
	}
}

func NewWorkerPoolSync() *WorkerPoolSync {
	return &WorkerPoolSync{}
}

type WorkerPoolSync struct{}

// Go implements WorkerPool.
func (w *WorkerPoolSync) Go(task Task) {
	task()
}
