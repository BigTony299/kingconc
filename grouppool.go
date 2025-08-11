package kingconc

func NewGroupPool(c GroupPoolConfig) GroupPool {
	var wp WorkerPool

	if limit, ok := c.Limit.Get(); ok {
		wp = NewWorkerPoolStatic(limit)
	} else {
		wp = NewWorkerPoolDynamic()
	}

	return GroupPool{
		workerPool: wp,
	}
}

type GroupPool struct {
	workerPool WorkerPool
}

func (p *GroupPool) Go(tasks ...TaskErr) <-chan error {
	errors := make(chan error, len(tasks))

	if len(tasks) == 0 {
		close(errors)
		return errors
	}

	counter := NewAtomicNumber(0)

	when := func(i int) bool { return i >= len(tasks) }
	then := func() { close(errors) }

	counter.Once(when, then)

	for _, task := range tasks {
		p.workerPool.Go(func() {
			select {
			case errors <- task():
				counter.Increment()
			default:
			}
		})
	}

	return errors
}

type GroupPoolConfig struct {
	Limit Option[int]
}
