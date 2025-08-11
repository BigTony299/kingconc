package kingconc_test

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/BigTony299/kingconc"
	"github.com/stretchr/testify/assert"
)

func TestGroupPool_MixedErrorSuccess(t *testing.T) {
	testCases := []struct {
		name   string
		config kingconc.GroupPoolConfig
	}{
		{"Static", kingconc.GroupPoolConfig{Limit: kingconc.Some(4)}},
		{"Dynamic", kingconc.GroupPoolConfig{Limit: kingconc.None[int]()}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := kingconc.NewGroupPool(tc.config)

			t.Run("EmptyTasks", func(t *testing.T) {
				errorChan := pool.Go()
				results := collectErrors(errorChan)
				assert.Len(t, results, 0)
			})

			t.Run("AllSuccess", func(t *testing.T) {
				tasks := make([]kingconc.TaskErr, 5)
				for i := range tasks {
					tasks[i] = func() error { return nil }
				}
				errorChan := pool.Go(tasks...)
				results := collectErrors(errorChan)
				assert.Len(t, results, 5)
				for _, err := range results {
					assert.NoError(t, err)
				}
			})

			t.Run("AllErrors", func(t *testing.T) {
				tasks := []kingconc.TaskErr{
					func() error { return errors.New("error1") },
					func() error { return errors.New("error2") },
					func() error { return errors.New("error3") },
				}
				errorChan := pool.Go(tasks...)
				results := collectErrors(errorChan)
				assert.Len(t, results, 3)
				for _, err := range results {
					assert.Error(t, err)
				}
			})

			t.Run("MixedResults", func(t *testing.T) {
				tasks := []kingconc.TaskErr{
					func() error { return nil },
					func() error { return errors.New("error1") },
					func() error { return nil },
					func() error { return errors.New("error2") },
				}
				errorChan := pool.Go(tasks...)
				results := collectErrors(errorChan)
				assert.Len(t, results, 4)

				nilCount := 0
				errCount := 0
				for _, err := range results {
					if err == nil {
						nilCount++
					} else {
						errCount++
					}
				}
				assert.Equal(t, 2, nilCount)
				assert.Equal(t, 2, errCount)
			})
		})
	}
}

func TestGroupPool_Concurrency(t *testing.T) {
	testCases := []struct {
		name   string
		config kingconc.GroupPoolConfig
	}{
		{"Static", kingconc.GroupPoolConfig{Limit: kingconc.Some(4)}},
		{"Dynamic", kingconc.GroupPoolConfig{Limit: kingconc.None[int]()}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := kingconc.NewGroupPool(tc.config)

			t.Run("HighConcurrency", func(t *testing.T) {
				tasks := make([]kingconc.TaskErr, 100)
				for i := range tasks {
					tasks[i] = func() error {
						time.Sleep(time.Microsecond * time.Duration(rand.Intn(100)))
						return nil
					}
				}
				errorChan := pool.Go(tasks...)
				results := collectErrors(errorChan)
				assert.Len(t, results, 100)
			})

			t.Run("ConcurrentWithErrors", func(t *testing.T) {
				tasks := make([]kingconc.TaskErr, 50)
				for i := range tasks {
					if i%2 == 0 {
						tasks[i] = func() error {
							time.Sleep(time.Microsecond * time.Duration(rand.Intn(50)))
							return nil
						}
					} else {
						tasks[i] = func() error {
							time.Sleep(time.Microsecond * time.Duration(rand.Intn(50)))
							return errors.New("concurrent error")
						}
					}
				}
				errorChan := pool.Go(tasks...)
				results := collectErrors(errorChan)
				assert.Len(t, results, 50)
			})

			t.Run("RaceConditionStress", func(t *testing.T) {
				tasks := make([]kingconc.TaskErr, 200)
				for i := range tasks {
					tasks[i] = func() error { return nil }
				}
				errorChan := pool.Go(tasks...)
				results := collectErrors(errorChan)
				assert.Len(t, results, 200)
			})
		})
	}
}

func collectErrors(errorChan <-chan error) []error {
	var results []error
	timeout := time.After(1 * time.Second)
	for {
		select {
		case err, ok := <-errorChan:
			if !ok {
				return results
			}
			results = append(results, err)
		case <-timeout:
			return results
		}
	}
}
