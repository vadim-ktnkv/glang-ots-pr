package hw06pipelineexecution

import "sync"

type (
	In  = <-chan interface{} //Read Only
	Out = In                 //Read Only
	Bi  = chan interface{}   //bi-directional
)

type Stage func(in In) (out Out)

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	//Stage executor
	var wg sync.WaitGroup
	wg.Add(len(in))

	stagesResult := make([]Out, len(in))
	output := make(Bi, len(in))
	index := 0
	for data := range in {
		input := make(Bi, 1)
		input <- data
		close(input)

		go func(inbound In, i int) {
			defer func() {
				stagesResult[i] = inbound
				wg.Done()
			}()

			for _, stage := range stages {
				inbound = stage(inbound)
			}
		}(input, index)
		index++
	}
	wg.Wait()

	for _, r := range stagesResult {
		output <- <-r
	}

	return output
}
