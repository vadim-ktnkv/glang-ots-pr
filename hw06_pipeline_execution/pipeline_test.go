package hw06pipelineexecution

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

const (
	sleepPerStage = time.Millisecond * 100
	fault         = sleepPerStage / 2
)

var isFullTesting = true

// Stage generator.
var g = func(_ string, f func(v interface{}) interface{}) Stage {
	return func(in In) Out {
		out := make(Bi)
		go func() {
			defer close(out)
			for v := range in {
				time.Sleep(sleepPerStage)
				out <- f(v)
			}
		}()
		return out
	}
}

var stages = []Stage{
	g("Dummy", func(v interface{}) interface{} { return v }),
	g("Multiplier (* 2)", func(v interface{}) interface{} { return v.(int) * 2 }),
	g("Adder (+ 100)", func(v interface{}) interface{} { return v.(int) + 100 }),
	g("Stringifier", func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
}

func TestPipeline(t *testing.T) {
	// Stage generator
	g := func(_ string, f func(v interface{}) interface{}) Stage {
		return func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					// fmt.Printf("will run %s with data %s\n", N, v)
					time.Sleep(sleepPerStage)
					out <- f(v)
				}
			}()
			return out
		}
	}

	stages := []Stage{
		g("Dummy", func(v interface{}) interface{} { return v }),
		g("Multiplier (* 2)", func(v interface{}) interface{} { return v.(int) * 2 }),
		g("Adder (+ 100)", func(v interface{}) interface{} { return v.(int) + 100 }),
		g("Stringifier", func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
	}

	t.Run("simple case", func(t *testing.T) {
		in := make(Bi)
		data := []int{1, 2, 3, 4, 5}

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]string, 0, 10)
		start := time.Now()
		for s := range ExecutePipeline(in, nil, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Equal(t, []string{"102", "104", "106", "108", "110"}, result)
		require.Less(t,
			int64(elapsed),
			// ~0.8s for processing 5 values in 4 stages (100ms every) concurrently
			int64(sleepPerStage)*int64(len(stages)+len(data)-1)+int64(fault))
	})

	t.Run("done case", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)
		data := []int{1, 2, 3, 4, 5}

		// Abort after 200ms
		abortDur := sleepPerStage * 2
		go func() {
			<-time.After(abortDur)
			close(done)
		}()

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]string, 0, 10)
		start := time.Now()
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Len(t, result, 0)
		require.Less(t, int64(elapsed), int64(abortDur)+int64(fault))
	})
}

func TestAllStageStop(t *testing.T) {
	if !isFullTesting {
		return
	}
	wg := sync.WaitGroup{}
	// Stage generator
	g := func(_ string, f func(v interface{}) interface{}) Stage {
		return func(in In) Out {
			out := make(Bi)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(out)
				for v := range in {
					time.Sleep(sleepPerStage)
					out <- f(v)
				}
			}()
			return out
		}
	}

	stages := []Stage{
		g("Dummy", func(v interface{}) interface{} { return v }),
		g("Multiplier (* 2)", func(v interface{}) interface{} { return v.(int) * 2 }),
		g("Adder (+ 100)", func(v interface{}) interface{} { return v.(int) + 100 }),
		g("Stringifier", func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
	}

	t.Run("done case", func(t *testing.T) {
		in := make(Bi)
		done := make(Bi)
		data := []int{1, 2, 3, 4, 5}

		// Abort after 200ms
		abortDur := sleepPerStage * 2
		go func() {
			<-time.After(abortDur)
			close(done)
		}()

		go func() {
			for _, v := range data {
				in <- v
			}
			close(in)
		}()

		result := make([]string, 0, 10)
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}
		wg.Wait()

		require.Len(t, result, 0)
	})
}

func TestAdditional(t *testing.T) {
	t.Run("routines leaks", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		data := []int{1, 2, 3, 4, 5}
		in := make(Bi, len(data))
		for _, v := range data {
			in <- v
		}
		close(in)

		done := make(Bi)
		// Abort after 200ms
		abortDur := sleepPerStage * 2
		go func() {
			<-time.After(abortDur)
			close(done)
		}()

		result := make([]string, 0, len(data))
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}

		require.Equal(t, []string{}, result)
	})

	t.Run("nil stage & nil data", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		data := []interface{}{1, 2, 3, 4, 5}
		in := make(Bi, len(data))
		for _, v := range data {
			in <- v
		}
		close(in)
		stash := stages[2]
		stages[2] = nil
		require.Panics(t, func() {
			ExecutePipeline(in, nil, stages...)
		})
		stages[2] = stash

		data[2] = nil
		in = make(Bi, len(data))
		for _, v := range data {
			in <- v
		}
		require.Panics(t, func() {
			ExecutePipeline(in, nil, stages...)
		})
	})
}

func BenchmarkXxx(b *testing.B) {
	data := []int{1, 2, 3, 4, 5}
	in := make(Bi, len(data))
	for _, v := range data {
		in <- v
	}
	close(in)
	b.Run("Bench", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			in = make(Bi, len(data))
			for _, v := range data {
				in <- v
			}
			close(in)
			b.StartTimer()

			ExecutePipeline(in, nil, stages...)
		}
	})
}
