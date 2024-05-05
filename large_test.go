package largecache

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWriteAndGetOnCache(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), DefaultConf(5*time.Second))
	value := []byte("value")

	cache.Set("key", value)
	cacheValue, err := cache.Get("key")

	noError(t, err)
	assertEqual(t, value, cacheValue)
}

func TestAppendAndGetOnCache(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), DefaultConf(5*time.Second))
	key := "key"
	value1 := make([]byte, 50)
	rand.Read(value1)
	value2 := make([]byte, 50)
	rand.Read(value2)
	value3 := make([]byte, 50)
	rand.Read(value3)

	_, err := cache.Get(key)

	assertEqual(t, ErrCannotRetrieveEntry, err)

	cache.Append(key, value1)
	cacheValue, err := cache.Get(key)

	noError(t, err)
	assertEqual(t, value1, cacheValue)

	cache.Append(key, value2)
	cacheValue, err = cache.Get(key)

	noError(t, err)
	expectedValue := value1
	expectedValue = append(expectedValue, value2...)
	assertEqual(t, expectedValue, cacheValue)

	cache.Append(key, value3)
	cacheValue, err = cache.Get(key)

	noError(t, err)
	expectedValue = value1
	expectedValue = append(expectedValue, value2...)
	expectedValue = append(expectedValue, value3...)
	assertEqual(t, expectedValue, cacheValue)
}

func TestAppendRandomly(t *testing.T) {
	t.Parallel()

	c := Config{
		Shards:             1,
		LifeWindow:         5 * time.Second,
		CleanWindow:        1 * time.Second,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntriesSize:     500,
		StatsEnabled:       true,
		Verbose:            true,
		Hasher:             newDefaultHasher(),
		HardMaxCacheSize:   1,
		Logger:             DefaultLogger(),
	}
	cache, err := New(context.Background(), c)
	noError(t, err)

	nKeys := 5
	nAppendsPerKey := 2000
	nWorker := 10
	var keys []string
	for i := 0; i < nAppendsPerKey; j++ {
		keys = append(keys, fmt.Sprintf("key%d", i))
	}
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	jobs := make(chan string, len(keys))
	for _, key := range keys {
		jobs <- key
	}
	close(jobs)

	var wg sync.WaitGroup
	for i := 0; i < nWorker; i++ {
		wg.Add(1)
		go func ()  {
			for {
				key, ok := <-jobs
				if !ok {
					break
				}
				cache.Append(key, []byte(key))
			}	
			wg.Done()
		}()
	}
	wg.Wait()

	assertEqual(t, nKeys, cache.Len())
	for i := 0; i < nKeys; i++ {
		for j := 0; j < nAppendsPerKey; j++ {
			keys = append(keys, fmt.Sprintf("key%d", i))
		}
	}

	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[rand.Int()
	})

	jobs := make(chan string, len(keys))
	for _, key := range keys {
		jobs <- key
	}
	close(jobs)

	var wg sync.WaitGroup
	for i := 0; i < nWorker; i++ {
		wg.Add(1)
		go func ()  {
			for {
				key, ok := <- jobs
				if !ok {
					break
				}
				cache.Append(key, []byte(key))
			}	
			wg.Done()
		}()
	}
	wg.Wait()

	assertEqual(t, nKeys, cache.Len())
	for i := 0; i < nKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		expectedValue := []byte(strings.Repeat(key, nAppendsPerKey))
		cacheValue, err := cache.Get(key)
		noError(t, err)
		assertEqual(t, expectedValue, cacheValue)
	}
}
