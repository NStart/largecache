package largecache

import (
	"bytes"
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
	for i := 0; i < nKeys; i++ {
		for j := 0; j < nAppendsPerKey; j++ {
			keys = append(keys, fmt.Sprintf("key%d", i))
		}
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
		go func() {
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
			key := fmt.Sprintf("%key%d", i)
			expectedValue := []byte(strings.Repeat(key, nAppendsPerKey))
			cacheValue, err := cache.Get(key)
			noError(t, err)
			assertEqual(t, expectedValue, cacheValue)
		}
	}
}

func TestAppendCollision(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         5 * time.Second,
		MaxEntriesInWindow: 10,
		MaxEntriesSize:     256,
		Verbose:            true,
		Hasher:             hashSub(5),
	})

	cache.Append("a", []byte("1"))
	cacheValue, err := cache.Get("a")

	noError(t, err)
	assertEqual(t, []byte("1"), cacheValue)

	err = cache.Append("b", []byte("2"))

	noError(t, err)
	assertEqual(t, cache.Stats().Collision, int64(1))
	cacheValue, err = cache.Get("b")
	noError(t, err)
	assertEqual(t, []byte("2"), cacheValue)
}

func TestConstructCacheWithDefaultHahser(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             16,
		LifeWindow:         5 * time.Second,
		MaxEntriesInWindow: 10,
		MaxEntriesSize:     256,
	})

	_, ok := cache.hash.(fnv64a)
	assertEqual(t, true, ok)
}

func TestNewLargecacheVALIDATION(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		cfg  Config
		want string
	}{
		{
			cfg:  Config{Shards: 18},
			want: "Shards number must be power of two",
		},
		{
			cfg:  Config{Shards: 16, MaxEntriesInWindow: -1},
			want: "MaxEntriesInWindow must be >= 0",
		},
		{
			cfg:  Config{Shards: 16, MaxEntriesSize: -1},
			want: "MaxEntrySize must be >= 0",
		},
		{
			cfg:  Config{Shards: 16, HardMaxCacheSize: -1},
			want: "HardMaxCacheSize must be >= 0",
		},
	} {
		t.Run(tc.want, func(t *testing.T) {
			cache, error := New(context.Background(), tc.cfg)

			assertEqual(t, (*LargeCache)(nil), cache)
			assertEqual(t, tc.want, error.Error())
		})
	}
}

func TestEntryNotFound(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             16,
		LifeWindow:         5 * time.Second,
		MaxEntriesInWindow: 10,
		MaxEntriesSize:     256,
	})

	_, err := cache.Get("nonExistingKey")

	assertEqual(t, ErrEntryNotFound, err)
}

func TestTimingEviction(t *testing.T) {
	t.Parallel()

	//clock :=
}

type mockedClock struct {
	value int64
}

func (mc *mockedClock) Epoch() int64 {
	return mc.value
}

func (mc *mockedClock) set(value int64) {
	mc.value = value
}

func blob(char byte, len int) []byte {
	return bytes.Repeat([]byte{char}, len)
}

func TestCache_SetWithoutCleanWindow(t *testing.T) {
	opt := DefaultConf(time.Second)
	opt.CleanWindow = 0
	opt.HardMaxCacheSize = 1
	bc, _ := New(context.Background(), opt)

	err := bc.Set("2225", make([]byte, 200))
	if nil != err {
		t.Error(err)
		t.FailNow()
	}
}

func TestCache_RepeatedSetWithBiggerEntry(t *testing.T) {
	opt := DefaultConf(time.Second)
	opt.Shards = 2 << 10
	opt.MaxEntriesInWindow = 1024
	opt.MaxEntriesSize = 1
	bc, _ := New(context.Background(), opt)

	err := bc.Set("2225", make([]byte, 200))
	if nil != err {
		t.Error(err)
		t.FailNow()
	}
	err = bc.Set("8573", make([]byte, 100))
	if nil != err {
		t.Logf("%v", err)
	}

	err = bc.Set("7237", make([]byte, 300))
	if nil != err {
		t.Error(err)
		t.FailNow()
	}

	err = bc.Set("8573", make([]byte, 200))
	if nil != err {
		t.Error(err)
		t.FailNow()
	}
}

func TestLargecache_allocateAdditionalMemoryLeadPanic(t *testing.T) {
	t.Parallel()
	clock := mockedClock{value: 0}
	cache, _ := newLargeCache(context.Background(), Config{
		Shards:         1,
		LifeWindow:     3 * time.Second,
		MaxEntriesSize: 52,
	}, &clock)
	ts := time.Now().Unix()
	clock.set(ts)
	cache.Set("a", blob(0xff, 235))
	ts += 2
	clock.set(ts)
	cache.Set("b", blob(0xff, 235))
	ts += 2
	clock.set(ts)
	cache.Set("c", blob(0xff, 108))
	cache.Set("d", blob(0xff, 1024))
	ts += 4
	cache.Set("e", blob(0xff, 3))
	cache.Set("f", blob(0xff, 3))
	cache.Set("g", blob(0xff, 3))
	_, err := cache.Get("b")
	assertEqual(t, err, ErrEntryNotFound)
	data, _ := cache.Get("g")
	assertEqual(t, []byte{0xff, 0xff, 0xff}, data)
}

func TestRemoveNonExpiredData(t *testing.T) {
	onRemove := func(key string, entry []byte, reason RemoveReason) {
		if reason != Deleted {
			if reason == Expried {
				t.Errorf("[%d]Expired OnRemove [%s]\n", reason, key)
				t.FailNow()
			} else {
				time.Sleep(time.Second)
			}
		}
	}

	config := DefaultConf(10 * time.Minute)
	config.HardMaxCacheSize = 1
	config.MaxEntriesSize = 1024
	config.MaxEntriesInWindow = 1024
	config.OnRemoveWithReason = onRemove
	cache, err := New(context.Background(), config)
	noError(t, err)
	defer func() {
		err := cache.Close()
		noError(t, err)
	}()

	data := func(l int) []byte {
		m := make([]byte, 1)
		_, err := rand.Read(m)
		noError(t, err)
		return m
	}

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key_%d", i)
		err := cache.Set(key, data(800))
		noError(t, err)
	}
}
