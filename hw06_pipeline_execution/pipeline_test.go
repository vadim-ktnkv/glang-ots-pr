package hw06pipelineexecution

import (
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
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

func TestPipeline(t *testing.T) {
	// Stage generator
	g := func(_ string, f func(v interface{}) interface{}) Stage {
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
	g := func(_ string, f func(v interface{}) interface{}) Stage {
		return func(in In) Out {
			out := make(Bi)
			go func() {
				defer close(out)
				for v := range in {
					sleepfor := rand.Intn(30) + 30
					time.Sleep(time.Millisecond * time.Duration(sleepfor))
					out <- f(v)
				}
			}()
			return out
		}
	}
	t.Run("routines leaks", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		stages := []Stage{
			g("Dummy", func(v interface{}) interface{} { return v }),
			g("Multiplier (* 2)", func(v interface{}) interface{} { return v.(int) * 2 }),
			g("Adder (+ 100)", func(v interface{}) interface{} { return v.(int) + 100 }),
			g("Stringifier", func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
		}
		data := []int{1, 2, 3, 4, 5}
		in := make(Bi, len(data))
		for _, v := range data {
			in <- v
		}
		close(in)

		result := make([]string, 0, len(data))
		for s := range ExecutePipeline(in, nil, stages...) {
			result = append(result, s.(string))
		}

		require.Equal(t, []string{"102", "104", "106", "108", "110"}, result)
	})

	t.Run("nil stage", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		dummyCount := atomic.Int32{}
		MultiplierCount := atomic.Int32{}
		AdderCount := atomic.Int32{}
		StringifierCount := atomic.Int32{}

		stages := []Stage{
			g("Dummy", func(v interface{}) interface{} { dummyCount.Add(1); return v }),
			g("Multiplier (* 2)", func(v interface{}) interface{} { MultiplierCount.Add(1); return v.(int) * 2 }),
			g("Adder (+ 100)", func(v interface{}) interface{} { AdderCount.Add(1); return v.(int) + 100 }),
			g("Stringifier", func(v interface{}) interface{} { StringifierCount.Add(1); return strconv.Itoa(v.(int)) }),
		}

		data := []interface{}{1, 2, 3, 4, 5}
		in := make(Bi, len(data))
		for _, v := range data {
			in <- v
		}
		close(in)

		stages[2] = nil
		dummyExpected := int32(len(data))
		MultiplierExpected := int32(len(data))
		AdderExpected := int32(0)
		StringifierExpected := int32(0)

		result := make([]interface{}, 0, len(data))
		for s := range ExecutePipeline(in, nil, stages...) {
			result = append(result, s)
		}

		required := make([]interface{}, 0, len(data))
		for range len(data) {
			required = append(required, errNilStage)
		}

		require.Equal(t, required, result)
		require.Equal(t, dummyExpected, dummyCount.Load())
		require.Equal(t, MultiplierExpected, MultiplierCount.Load())
		require.Equal(t, AdderExpected, AdderCount.Load())
		require.Equal(t, StringifierExpected, StringifierCount.Load())
	})
}
