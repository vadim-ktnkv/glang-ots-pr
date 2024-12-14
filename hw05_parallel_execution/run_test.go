package hw05parallelexecution

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func produceSleepTasks(tasksCount int) ([]Task, *atomic.Int32) {
	tasks := make([]Task, 0, tasksCount)
	var runTasksCount atomic.Int32
	for i := 0; i < tasksCount; i++ {
		tasks = append(tasks, func() error {
			defer runTasksCount.Add(1)
			time.Sleep(time.Microsecond * 20)
			return nil
		})
	}
	return tasks, &runTasksCount
}

func produceFibonacciTasks(tasksCount int, n uint, errorAt uint) ([]Task, *atomic.Int32) {
	tasks := make([]Task, 0, tasksCount)
	var runTasksCount atomic.Int32
	for i := 0; i < tasksCount; i++ {
		// calculate Fibonacci for n; and produce error at iteration "errorAt"
		tasks = append(tasks, func() error {
			defer runTasksCount.Add(1)
			if n <= 1 {
				return nil
			}
			if errorAt != 0 && errorAt < 2 {
				return fmt.Errorf("Error at iteration %d", errorAt)
			}
			prev, curr := uint(0), uint(1)
			for i := uint(2); i <= n; i++ {
				if errorAt == i {
					return fmt.Errorf("Error at iteration %d", i)
				}
				prev, curr = curr, curr+prev
			}
			return nil
		})
	}
	return tasks, &runTasksCount
}

func TestRun(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("if were errors in first M tasks, than finished not more N+M tasks", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			err := fmt.Errorf("error from task %d", i)
			tasks = append(tasks, func() error {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				atomic.AddInt32(&runTasksCount, 1)
				return err
			})
		}

		workersCount := 10
		maxErrorsCount := 23
		err := Run(tasks, workersCount, maxErrorsCount)

		require.Truef(t, errors.Is(err, ErrErrorsLimitExceeded), "actual err - %v", err)
		require.LessOrEqual(t, runTasksCount, int32(workersCount+maxErrorsCount), "extra tasks were started")
	})

	t.Run("tasks without errors", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)
		require.NoError(t, err)

		require.Equal(t, runTasksCount, int32(tasksCount), "not all tasks were completed")
		require.LessOrEqual(t, int64(elapsedTime), int64(sumTime/2), "tasks were run sequentially?")
	})
}

func TestRunAdditional(t *testing.T) {
	defer goleak.VerifyNone(t)
	t.Run("0 workers", func(t *testing.T) {
		tasksCount := 5
		workersCount := 0
		maxErrorsCount := 0
		fibonacciNum := uint(rand.Intn(10) + 500_000) // non-constant due to unparam

		tasks, runTasksCount := produceFibonacciTasks(tasksCount, fibonacciNum, 0)
		err := Run(tasks, workersCount, maxErrorsCount)

		require.ErrorIs(t, err, ErrWorkersCountLow, "actual err - %v", err)
		require.Equal(t, int32(0), runTasksCount.Load(), "no tasks must be completed")
	})

	t.Run("-20 workers", func(t *testing.T) {
		tasksCount := 5
		workersCount := -20
		maxErrorsCount := 0
		fibonacciNum := uint(rand.Intn(10) + 500_000) // non-constant due to unparam

		tasks, runTasksCount := produceFibonacciTasks(tasksCount, fibonacciNum, 0)
		err := Run(tasks, workersCount, maxErrorsCount)

		require.ErrorIs(t, err, ErrWorkersCountLow, "actual err - %v", err)
		require.Equal(t, int32(0), runTasksCount.Load(), "no tasks must be completed")
	})

	t.Run("half is nil tasks", func(t *testing.T) {
		tasksCount := 10
		workersCount := 3
		maxErrorsCount := 0
		fibonacciNum := uint(rand.Intn(10) + 500_000) // non-constant due to unparam

		tasks, runTasksCount := produceFibonacciTasks(tasksCount, fibonacciNum, 0)
		countNilled := 0
		for i := range tasksCount / 2 {
			tasks[i*2] = nil
			countNilled++
		}
		err := Run(tasks, workersCount, maxErrorsCount)
		require.NoError(t, err)
		require.Equal(t, int32(tasksCount-countNilled), runTasksCount.Load(), "not all tasks were completed")
	})

	t.Run("ignore errors check, maxErrorsCount = 0", func(t *testing.T) {
		tasksCount := 10
		workersCount := 3
		maxErrorsCount := 0
		fibonacciNum := uint(20)

		tasks, runTasksCount := produceFibonacciTasks(tasksCount, fibonacciNum, 1)
		err := Run(tasks, workersCount, maxErrorsCount)

		// Assume that maxErrorsCount = 0 implies ignore errors.
		require.NoError(t, err, "tasks must complete without errors")
		require.Equal(t, int32(tasksCount), runTasksCount.Load(), "not all tasks were completed")
	})

	t.Run("ignore errors check,maxErrorsCount = -10", func(t *testing.T) {
		tasksCount := 10
		workersCount := 3
		maxErrorsCount := -10
		fibonacciNum := uint(20)

		tasks, runTasksCount := produceFibonacciTasks(tasksCount, fibonacciNum, 1)
		err := Run(tasks, workersCount, maxErrorsCount)

		require.NoError(t, err, "tasks must complete without errors")
		require.Equal(t, int32(tasksCount), runTasksCount.Load(), "not all tasks were completed")
	})

	t.Run("workers more then tasks, no time.Sleep", func(t *testing.T) {
		tasksCount := 10
		workersCount := 50
		maxErrorsCount := 0
		fibonacciNum := uint(rand.Intn(10) + 500_000) // non-constant due to unparam

		tasks, runTasksCount := produceFibonacciTasks(tasksCount, fibonacciNum, 0)

		err := Run(tasks, workersCount, maxErrorsCount)

		require.Equal(t, int32(tasksCount), runTasksCount.Load(), "not all tasks were completed")
		require.NoError(t, err, "tasks must complete without errors")
	})

	t.Run("test concurrency without sleep", func(t *testing.T) {
		sampleCount := 5
		maxErrorsCount := 1
		fibonacciNum := uint(rand.Intn(10) + 500_000) // non-constant due to unparam

		// sampling average time for one task
		tasks, runTasksCount := produceFibonacciTasks(1, fibonacciNum, 0)
		var samples int64
		for range sampleCount {
			start := time.Now()
			Run(tasks, 1, maxErrorsCount)
			samples += time.Since(start).Microseconds()
		}
		require.Eventually(t, func() bool {
			return samples != 0
		}, time.Second*5, time.Millisecond*100, "Sampling took too long")
		require.Equal(t, int32(sampleCount), runTasksCount.Load(), "not all tasks were completed")
		require.NotEqual(t, int64(0), samples/int64(sampleCount), "cant sample execution time")

		workersCount := runtime.GOMAXPROCS(-1) // use all available threads
		tasksCount := 50
		seqExecTime := (samples / int64(sampleCount)) * int64(tasksCount)
		tasks, runTasksCount = produceFibonacciTasks(tasksCount, fibonacciNum, 0)
		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)

		require.Eventually(t, func() bool {
			return elapsedTime != 0
		}, time.Second*10, time.Millisecond*100, "Tasks took too long")
		require.NoError(t, err, "tasks must complete without errors")
		require.Equal(t, int32(tasksCount), runTasksCount.Load(), "not all tasks were completed")
		require.LessOrEqual(t, elapsedTime.Microseconds(), seqExecTime/2, "tasks were run sequentially?")
	})
}

func BenchmarkTasks(b *testing.B) {
	tasksCount := 100
	errorsAllowed := 10
	var tasks []Task

	workerCount := runtime.GOMAXPROCS(-1)
	tasks, _ = produceSleepTasks(tasksCount)

	b.Run("BenchSleep", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Run(tasks, workerCount, errorsAllowed)
		}
	})

	tasks, _ = produceFibonacciTasks(tasksCount, 500_000, 0)
	b.Run("BenchFibonacci", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Run(tasks, workerCount, errorsAllowed)
		}
	})
}
