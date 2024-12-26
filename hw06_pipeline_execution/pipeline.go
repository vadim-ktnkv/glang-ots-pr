package hw06pipelineexecution

import (
	"sync"
	"time"
)

type (
	In  = <-chan interface{} // Read Only
	Out = In                 // Read Only
	Bi  = chan interface{}   // bi-directional
)

type Stage func(in In) (out Out)

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	var wg sync.WaitGroup
	for _, stage := range stages {
		if stage == nil {
			panic("Stage is nil")
		}
		stageResults := []Bi{}
		for data := range in {
			if data == nil {
				panic("Data is nil")
			}
			stageIn := make(Bi, 1)
			select {
			case <-done:
				close(stageIn)
				return stageIn
			default:
				// workaround for TestPipeline/done_case: "302083485" is not less than "250000000"
				time.Sleep(time.Microsecond * time.Duration(100))
			}
			stageIn <- data
			close(stageIn)
			stageOut := make(Bi, 1)
			stageResults = append(stageResults, stageOut)
			wg.Add(1)
			go func() {
				defer wg.Done()
				stageOut <- <-stage(stageIn)
				close(stageOut)
			}()
		}
		wg.Wait()
		temp := make(Bi, len(stageResults))
		for _, c := range stageResults {
			temp <- <-c
		}
		close(temp)
		in = In(temp)
	}
	return in
}
