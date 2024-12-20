package hw06pipelineexecution

import (
	"sync"
)

type (
	In  = <-chan interface{} // Read Only
	Out = In                 // Read Only
	Bi  = chan interface{}   // bi-directional
)

type Stage func(in In) (out Out)

func isTerminate(done In) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	// Stage executor
	var wg sync.WaitGroup
	stagesResult := []Bi{}
	output := make(Bi)
	// Process input data in parallel
	for data := range in {
		pipelineOut := make(Bi, 1)
		pipelineIn := make(Bi, 1)
		pipelineIn <- data
		close(pipelineIn)
		// Executing pipeline
		go func(inbound In, outbound Bi) {
			defer func() {
				outbound <- <-inbound
				close(outbound)
				wg.Done()
			}()
			for _, stage := range stages {
				inbound = stage(inbound)
			}
		}(pipelineIn, pipelineOut)
		/////////////////////////////

		stagesResult = append(stagesResult, pipelineOut)
		wg.Add(1)
	}

	wg.Wait()
	if isTerminate(done) {
		close(output)
	} else {
		go func() {
			for _, data := range stagesResult {
				output <- <-data
			}
			close(output)
		}()
	}

	return output
}
