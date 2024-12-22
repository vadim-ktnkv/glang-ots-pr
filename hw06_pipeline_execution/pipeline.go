package hw06pipelineexecution

import (
	"sync"
	"sync/atomic"
	"time"
)

type (
	In  = <-chan interface{} // Read Only
	Out = In                 // Read Only
	Bi  = chan interface{}   // bi-directional
)

var terminate atomic.Bool

type Stage func(in In) (out Out)

func PipelineExecutor(wg *sync.WaitGroup, stages []Stage, input In, output Bi) {
	defer func() {
		output <- <-input
		close(output)
		wg.Done()
	}()

	for _, stage := range stages {
		if terminate.Load() {
			return
		}
		temp := make(Bi, 1)
		temp <- <-stage(input)
		close(temp)
		input = temp
		// Workaround:
		// Tried fix "TestPipeline/done_case" failure: "302813589" is not less than "250000000",
		// which appears 1 of ~30 test iteration, caused by late workers executed tasks from next Stages
		// Fixed it with skipping wg.Wait(), when <-done termination signal arrives.
		// This caused a RACING issue in test "TestAllStageStop/done_case".
		// The one-size-fits-all solution discovered: slow down workers by 1ms before next iteration,
		// it prevent next Stages tasks execution in case of termination signal arrives.
		time.Sleep(time.Millisecond)
	}
}

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	var wg sync.WaitGroup
	executionResults := []Bi{}
	terminate.Store(false)
	// This routine will set terminate signal when done is closed, notify workers to stop
	go func() {
		<-done
		terminate.Store(true)
	}()

	count := 0
	for data := range in {
		pipelineIn := make(Bi, 1)
		pipelineIn <- data
		close(pipelineIn)
		pipelineOut := make(Bi, 1)
		executionResults = append(executionResults, pipelineOut) // preserve sequence of input data for output
		go PipelineExecutor(&wg, stages, pipelineIn, pipelineOut)
		wg.Add(1)
		count++
	}
	output := make(Bi, count)
	wg.Wait()
	// In case if no termination signal, write result of the execution of the pipelines
	// to output chan, with the same sequence as input data
	if !terminate.Load() {
		for _, pipelineOut := range executionResults {
			output <- <-pipelineOut
		}
	}

	close(output)
	return output
}
