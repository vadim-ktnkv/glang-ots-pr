package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")
var suscessCount atomic.Int32
var errorsCount atomic.Int32

type Task func() error

func Worker(wg *sync.WaitGroup, workerNum int, tasks <-chan Task, m int) {
	defer wg.Done()
	fmt.Println("STARTED Worker #", workerNum)

	for task := range tasks {
		if int(errorsCount.Load()) >= m {
			break
		}
		error := task()
		if error != nil {
			fmt.Printf("Worker #%d: got error\n", workerNum)
			errorsCount.Add(1)
		} else {
			fmt.Printf("Worker #%d: reprot for compleated task, sum: %d\n", workerNum, suscessCount.Add(1))
		}
	}
	fmt.Println("Worker #", workerNum, " terminated")
}

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(tasks []Task, n, m int) error {
	jobs := make(chan Task, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for workerNum := range n {
		workerNum++
		go Worker(&wg, workerNum, jobs, m)
	}
	fmt.Println("Processing jobs")

	for taskNum := range len(tasks) {
		if tasks[taskNum] == nil {
			continue
		}

		if int(errorsCount.Load()) >= m {
			break
		}
		jobs <- tasks[taskNum]
	}
	close(jobs)
	wg.Wait()
	fmt.Printf("Finished with:\n Success: %d\n Errors: %d", suscessCount.Load(), errorsCount.Load())
	return nil
}
