package hw05parallelexecution

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

func WorkerAtomic(wg *sync.WaitGroup, _ int, tasksQueue <-chan Task, errorsCounter *atomic.Int32, m int) {
	defer wg.Done()
	// fmt.Println("STARTED Worker #", workerNum)

	for task := range tasksQueue {
		if int(errorsCounter.Load()) >= m {
			break
		}
		taskResult := task()
		if taskResult != nil {
			// fmt.Printf("Worker #%d: got error\n", workerNum)
			errorsCounter.Add(1)
		}
		// else {
		// 	fmt.Printf("Worker #%d: completed task, sum: %d\n", workerNum, successCount.Add(1))
		// }
	}
	// fmt.Println("TERMINATED Worker #", workerNum)
}

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func RunAtomic(inboundTasks []Task, n, m int) error {
	var errorsCounter atomic.Int32
	tasksQueue := make(chan Task, n)
	var wg sync.WaitGroup
	wg.Add(n)
	var returnValue error

	for workerNum := range n {
		workerNum++
		go WorkerAtomic(&wg, workerNum, tasksQueue, &errorsCounter, m)
	}
	// fmt.Println("Processing jobs")

	for _, currTask := range inboundTasks {
		if int(errorsCounter.Load()) >= m {
			returnValue = ErrErrorsLimitExceeded
			break
		}
		if currTask == nil {
			continue
		}
		tasksQueue <- currTask
	}
	close(tasksQueue)
	wg.Wait()
	return returnValue
}

func WorkerChan(wg *sync.WaitGroup, tasksQueue <-chan Task, errorsQueue chan<- struct{}) {
	defer wg.Done()

	for task := range tasksQueue {
		if task == nil {
			continue
		}
		taskResult := task()
		if taskResult != nil {
			select {
			case errorsQueue <- struct{}{}:
			default:
				return
			}
		}
	}
}

func TasksDispatcher(inboundTasks []Task, tasksQueue chan<- Task, closeDispatcher <-chan struct{}) {
	defer close(tasksQueue)

	for _, currTask := range inboundTasks {
		select {
		case tasksQueue <- currTask:
		case <-closeDispatcher:
			return
		}
	}
}

func Run(inboundTasks []Task, n, m int) error {
	tasksQueue := make(chan Task, n)
	errorsQueue := make(chan struct{}, m)
	closeDispatcher := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go WorkerChan(&wg, tasksQueue, errorsQueue)
	}

	go TasksDispatcher(inboundTasks, tasksQueue, closeDispatcher)
	wg.Wait()
	close(closeDispatcher)
	close(errorsQueue)

	if len(errorsQueue) == m {
		return ErrErrorsLimitExceeded
	}
	return nil
}
