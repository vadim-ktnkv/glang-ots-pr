package hw05parallelexecution

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrErrorsLimitExceeded = errors.New("errors limit exceeded")
	successCount           atomic.Int32
	errorsCount            atomic.Int32
)

type Task func() error

func WorkerAtomic(wg *sync.WaitGroup, _ int, tasks <-chan Task, m int) {
	defer wg.Done()
	// fmt.Println("STARTED Worker #", workerNum)

	for task := range tasks {
		if int(errorsCount.Load()) >= m {
			break
		}
		taskResult := task()
		if taskResult != nil {
			// fmt.Printf("Worker #%d: got error\n", workerNum)
			errorsCount.Add(1)
		}
		// else {
		// 	fmt.Printf("Worker #%d: completed task, sum: %d\n", workerNum, successCount.Add(1))
		// }
	}
	// fmt.Println("TERMINATED Worker #", workerNum)
}

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func RunAtomic(tasks []Task, n, m int) error {
	jobs := make(chan Task, n)
	var wg sync.WaitGroup
	wg.Add(n)

	successCount.Store(0)
	errorsCount.Store(0)

	for workerNum := range n {
		workerNum++
		go WorkerAtomic(&wg, workerNum, jobs, m)
	}
	// fmt.Println("Processing jobs")

	for taskNum := range len(tasks) {
		if tasks[taskNum] == nil {
			continue
		}

		if int(errorsCount.Load()) >= m {
			close(jobs)
			wg.Wait()
			// fmt.Printf("ER finished with: Success: %d Errors: %d\n", successCount.Load(), errorsCount.Load())
			return ErrErrorsLimitExceeded
		}
		jobs <- tasks[taskNum]
	}
	close(jobs)
	wg.Wait()
	// fmt.Printf("SC finished with: Success: %d Errors: %d\n", successCount.Load(), errorsCount.Load())
	return nil
}

func WorkerChan(wg *sync.WaitGroup, _ int, tasks <-chan Task, errors chan<- struct{}) {
	defer func() {
		// fmt.Println("TERMINATED Worker #", workerNum)
		wg.Done()
	}()

	// fmt.Println("STARTED Worker #", workerNum)

	for {
		task, ok := <-tasks
		if !ok {
			return
		}
		if task == nil {
			continue
		}
		execError := task()
		if execError != nil {
			// fmt.Println("Got error; Worker#", workerNum)
			select {
			case errors <- struct{}{}:
			default:
				return
			}
		}
		// else {
		// 	fmt.Println("Task successful, Worker#", workerNum)
		// }
	}
}

func Run(inboundTasks []Task, n, m int) error {
	tasks := make(chan Task, n)
	errors := make(chan struct{}, m)
	var result error
	var wg sync.WaitGroup
	wg.Add(n)

	for workerNum := range n {
		workerNum++
		go WorkerChan(&wg, workerNum, tasks, errors)
	}
	// fmt.Println("Processing jobs")

	taskNum := 0
	for {
		if len(errors) == m {
			result = ErrErrorsLimitExceeded
			break
		}
		if taskNum < len(inboundTasks) {
			tasks <- inboundTasks[taskNum]
			taskNum++
		} else {
			break
		}
	}
	close(tasks)
	wg.Wait()
	close(errors)

	return result
}
