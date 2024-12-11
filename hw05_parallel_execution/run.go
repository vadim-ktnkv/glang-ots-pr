package hw05parallelexecution

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

func WorkerAtomic(wg *sync.WaitGroup, inboundTasks []Task, errorsCounter *atomic.Int32, m, start, end int) {
	defer wg.Done()
	errorsMax := int32(m)
	var taskResult error
	for i := start; i < end; i++ {
		if errorsCounter.Load() >= errorsMax {
			break
		}
		if inboundTasks[i] == nil {
			continue
		}
		taskResult = inboundTasks[i]()
		if taskResult != nil {
			errorsCounter.Add(1)
		}
	}
}

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(inboundTasks []Task, n, m int) error {
	var wg sync.WaitGroup
	var errorsCounter atomic.Int32

	if m <= 0 {
		m = 1
	}

	tasksWorkersRatio := float64(len(inboundTasks)) / float64(n)
	if tasksWorkersRatio < 1 {
		n = len(inboundTasks)
	}
	wg.Add(n)

	tasksPerWorker := int(math.Ceil(tasksWorkersRatio))

	startPos := 0

	for range n {
		endPos := startPos + tasksPerWorker + 1
		if endPos > len(inboundTasks) {
			endPos = len(inboundTasks)
		}
		go WorkerAtomic(&wg, inboundTasks, &errorsCounter, m, startPos, endPos)
		startPos = endPos
	}

	wg.Wait()
	if errorsCounter.Load() >= int32(m) {
		return ErrErrorsLimitExceeded
	}
	return nil
}

func WorkerChan(wg *sync.WaitGroup, tasksQueue <-chan Task, errorsQueue chan<- struct{}) {
	defer wg.Done()
	var taskResult error
	for task := range tasksQueue {
		if task == nil {
			continue
		}
		taskResult = task()
		if taskResult != nil {
			select {
			case errorsQueue <- struct{}{}:
			default:
				return
			}
		}
	}
}

func TasksDispatcher(inboundTasks []Task, tasksQueue chan<- Task, closeSignal <-chan struct{}) {
	defer close(tasksQueue)

	for _, newTask := range inboundTasks {
		select {
		case <-closeSignal:
			return
		case tasksQueue <- newTask:
		}
	}
}

func RunChan(inboundTasks []Task, n, m int) error {
	tasksQueue := make(chan Task, len(inboundTasks))
	errorsQueue := make(chan struct{}, m)
	closeSignal := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	go TasksDispatcher(inboundTasks, tasksQueue, closeSignal)
	for range n {
		go WorkerChan(&wg, tasksQueue, errorsQueue)
	}
	wg.Wait()
	close(closeSignal)
	close(errorsQueue)

	if len(errorsQueue) == m {
		return ErrErrorsLimitExceeded
	}
	return nil
}
