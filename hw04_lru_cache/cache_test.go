package hw04lrucache

import (
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewCache(10)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		c := NewCache(5)

		wasInCache := c.Set("aaa", 100)
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200)
		require.False(t, wasInCache)

		val, ok := c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb")
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300)
		require.True(t, wasInCache)

		val, ok = c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc")
		require.False(t, ok)
		require.Nil(t, val)
	})

	var cacheCreated, cacheFilled, cacheCleared runtime.MemStats
	cacheCapacity := 1000
	keyLen := 6

	getRandomKey := func() Key {
		x := make([]byte, keyLen)
		for i := range keyLen {
			// get random char between ASCII #48 and #126(0-9, a-z, A-Z and some printable symbols)
			d := rand.Int31n(79) + 48
			x[i] = byte(d)
		}
		return Key(x)
	}

	t.Run("purge logic", func(t *testing.T) {
		c := NewCache(cacheCapacity)
		runtime.ReadMemStats(&cacheCreated)

		for i := range cacheCapacity {
			c.Set(getRandomKey(), (i*10)+i)
		}

		runtime.ReadMemStats(&cacheFilled)
		c.Clear()
		runtime.GC()
		runtime.ReadMemStats(&cacheCleared)
		result := (cacheFilled.HeapObjects - cacheCleared.HeapObjects) >= (cacheFilled.HeapObjects - cacheCreated.HeapObjects)
		require.True(t, result)
	})
}

func TestCacheMultithreading(t *testing.T) {
	// t.Skip() // Remove me if task with asterisk completed.

	c := NewCache(10)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Set(Key(strconv.Itoa(i)), i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Get(Key(strconv.Itoa(rand.Intn(1_000_000))))
		}
	}()

	wg.Wait()

	require.True(t, true)
}
