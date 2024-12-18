package hw05parallelexecution

import (
	"errors"
	"sync"
)

var (
	ErrErrorsLimitExceeded = errors.New("errors limit exceeded")
	ErrWorkersCountLow     = errors.New("workers count must be >0")
)

type Task func() error

func WorkerChan(wg *sync.WaitGroup, tasksQueue <-chan Task, errorsQueue chan<- struct{}, ignoreErrors bool) {
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
				if !ignoreErrors {
					return
				}
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

func Run(inboundTasks []Task, n, m int) error {
	tasksQueue := make(chan Task, len(inboundTasks))
	// There is no point in running anything if the number of workers is 0, so return error of invalid parameter
	if n <= 0 {
		return ErrWorkersCountLow
	}
	var ignoreErrors bool
	if m <= 0 {
		ignoreErrors = true
		m = 0
	}

	errorsQueue := make(chan struct{}, m)
	closeSignal := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	go TasksDispatcher(inboundTasks, tasksQueue, closeSignal)
	for range n {
		go WorkerChan(&wg, tasksQueue, errorsQueue, ignoreErrors)
	}
	wg.Wait()
	close(closeSignal)
	close(errorsQueue)

	if m > 0 && len(errorsQueue) == m {
		return ErrErrorsLimitExceeded
	}
	return nil
}
