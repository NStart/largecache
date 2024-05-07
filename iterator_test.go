package largecache

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestEntriesIterator(t *testing.T) {
	t.Parallel()

	keysCount := 1000
	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         6 * time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})
	value := []byte("value")
	for i := 0; i < keysCount; i++ {
		cache.Set(fmt.Sprintf("key%d", i), value)
	}

	keys := make(map[string]struct{})
	iterator := cache.Iterator()

	for iterator.SetNext() {
		current, err := iterator.Value()

		if err == nil {
			keys[current.key] = struct{}{}
		}
	}

	assertEqual(t, keysCount, len(keys))
}

func TestEntriesIteratorWithMostShardsEmpty(t *testing.T) {
	t.Parallel()

	clock := mockedClock{value: 0}
	cache, _ := newLargeCache(context.Background(), Config{
		Shards:             8,
		LifeWindow:         6 * time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	}, &clock)

	cache.Set("key", []byte("value"))

	iterator := cache.Iterator()

	if !iterator.SetNext() {
		t.Errorf("Iterator should contain at least single element")
	}

	current, err := iterator.Value()
	noError(t, err)
	assertEqual(t, "key", current.Key())
	assertEqual(t, uint64(0x3dc94a19365b10ec), current.Hash())
	assertEqual(t, []byte("value"), current.Value())
	assertEqual(t, uint64(0), current.Timestamp())
}

func TestEntriesIteratorWithConcurrentUpdate(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     1,
	})

	cache.Set("key", []byte("value"))

	iterator := cache.Iterator()

	if !iterator.SetNext() {
		t.Errorf("Iterator should contain at least single element")
	}

	getOldestEntry := func(s *cacheShard) ([]byte, error) {
		s.lock.RLock()
		defer s.lock.RUnlock()
		return s.entries.Peek()
	}

	for i := 0; i < cache.config.Shards; i++ {
		if oldestEntry, err := getOldestEntry(cache.shards[i]); err == nil {
			cache.onEvict(oldestEntry, 10, cache.shards[i].removeOldestEntry)
		}
	}

	current, err := iterator.Value()
	assertEqual(t, nil, err)
	assertEqual(t, []byte("value"), current.Value())

	next := iterator.SetNext()
	assertEqual(t, false, next)
}

func TestEntriesIteratorWithAllShardsEmpty(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})

	iterator := cache.Iterator()

	if iterator.SetNext() {
		t.Errorf("Iterator should not contact any elements")
	}
}

func TestEntriesIteratorInInvalidState(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})

	iterator := cache.Iterator()

	_, err := iterator.Value()
	assertEqual(t, ErrInvalidIteratorState, err)
	assertEqual(t, "Iterator is in invalid state. Use SetNext() to move to next position.", err.Error())
}

func TestEntriesIteratorParallelAdd(t *testing.T) {
	bc, err := New(context.Background(), DefaultConf(1*time.Minute))
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for i := 0; i < 10000; i++ {
			err := bc.Set(strconv.Itoa(i), []byte("aaaaaaa"))
			if err != nil {
				panic(err)
			}

			runtime.Gosched()
		}
		wg.Done()
	}()

	for i := 0; i < 100; i++ {
		iter := bc.Iterator()
		for iter.SetNext() {
			_, _ = iter.Value()
		}
	}
	wg.Wait()
}

func TestParallelSetAndIteration(t *testing.T) {
	t.Parallel()

	rand.Seed(0)

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 100,
		MaxEntriesSize:     256,
		HardMaxCacheSize:   1,
		Verbose:            true,
	})

	entrySize := 1024 * 100
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer func() {
			err := recover()
			assertEqual(t, err, nil)
		}()

		defer wg.Done()

		isTimeout := false
		for {
			if isTimeout {
				break
			}
			select {
			case <-ctx.Done():
				isTimeout = true
			default:
				err := cache.Set(strconv.Itoa(rand.Intn(100)), blob('a', entrySize))
				noError(t, err)
			}
		}
	}()

	go func() {
		defer func() {
			err := recover()
			assertEqual(t, nil, err)
		}()
		defer wg.Done()

		isTimeout := false
		for {
			if isTimeout {
				break
			}
			select {
			case <-ctx.Done():
				isTimeout = true
			default:
				iter := cache.Iterator()
				for iter.SetNext() {
					entry, err := iter.Value()

					noError(t, err)
					assertEqual(t, entrySize, len(entry.Value()))
				}
			}

		}
	}()

	wg.Wait()
}
