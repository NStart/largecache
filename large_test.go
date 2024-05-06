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

	clock := mockedClock{value: 0}
	cache, _ := newLargeCache(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	}, &clock)

	cache.Set(1)
	cache.Set("key2", []byte("value2"))
	_, err := cache.Get("key")

	noError(t, err)

	clock.set(5)
	cache.Set("key2", []byte("value2"))
	_, err = cache.Get("key")

	assertEqual(t, ErrEntryNotFound, err)
}

func TestTimintEvictionShouldEvctOnlyFromUpdatedShard(t *testing.T) {
	t.Parallel()

	clock := mockedClock{value: 0}
	cache, _ := newLargeCache(context.Background(), Config{
		Shards:             4,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	}, &clock)

	cache.Set("key", []byte("value"))
	clock.set(5)
	cache.Set("key2", []byte("value 2"))
	value, err := cache.Get("key")

	noError(t, err)
	assertEqual(t, []byte("value"), value)
}

func TestCleanShouldEvictAll(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             4,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})

	cache.Set("key", []byte("value"))
	<-time.After(3 * time.Second)
	value, err := cache.Get("key")

	assertEqual(t, ErrEntryNotFound, err)
	assertEqual(t, value, []byte(nil))
}

func TestOnRemoveCallBack(t *testing.T) {
	t.Parallel()

	clock := mockedClock{value: 0}
	onRemoveInvoked := false
	onRemoveExtInvoked := false
	onRemove := func(key string, entry []byte) {
		onRemoveInvoked = true
		assertEqual(t, "key", key)
		assertEqual(t, []byte("value"), entry)
	}

	onRemoveExt := func(key string, entry []byte, reason RemoveReason) {
		onRemoveExtInvoked = true
	}
	cache, _ := newLargeCache(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
		OnRemove:           onRemove,
		OnRemoveWithReason: onRemoveExt,
	}, &clock)

	cache.Set("key", []byte("value"))
	clock.set(5)
	cache.Set("key2", []byte("value2"))

	assertEqual(t, true, onRemoveInvoked)
	assertEqual(t, false, onRemoveExtInvoked)
}

func TestOnRemoveWithReasonCallBack(t *testing.T) {
	t.Parallel()

	clock := mockedClock{value: 0}
	onRemoveInvoked := false
	onRemove := func(key string, entry []byte, reason RemoveReason) {
		onRemoveInvoked = true
		assertEqual(t, "key", key)
		assertEqual(t, []byte("value"), entry)
		assertEqual(t, reason, RemoveReason(Expried))
	}
	cache, _ := newLargeCache(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
		OnRemoveWithReason: onRemove,
	}, &clock)

	cache.Set("key", []byte("value"))
	clock.set(5)
	cache.Set("key2", []byte("value2"))

	assertEqual(t, true, onRemoveInvoked)
}

func TestOnRemoveFilter(t *testing.T) {
	t.Parallel()

	clock := mockedClock{value: 0}
	onRemoveInvoked := false
	onRemove := func(key string, entry []byte, reason RemoveReason) {
		onRemoveInvoked = true
	}
	c := Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
		OnRemoveWithReason: onRemove,
	}

	cache, _ := newLargeCache(context.Background(), c, &clock)
	cache.Set("key", []byte("value"))
	clock.set(5)
	cache.Set("key2", []byte("value2"))

	assertEqual(t, false, onRemoveInvoked)

	cache.Delete("keys")

	assertEqual(t, true, onRemoveInvoked)
}

func TestOnRemoveFilterExpired(t *testing.T) {
	//t.Parallel()

	clock := mockedClock{value: 0}
	onRemoveDeleted, onRemoveExpired := false, false
	var err error
	onRemove := func(key string, entry []byte, reason RemoveReason) {
		switch reason {
		case Deleted:
			onRemoveDeleted = true
		case Expried:
			onRemoveExpired = true
		}
	}

	c := Config{
		Shards:             1,
		LifeWindow:         3 * time.Second,
		CleanWindow:        0,
		MaxEntriesInWindow: 10,
		MaxEntriesSize:     256,
		OnRemoveWithReason: onRemove,
	}

	cache, err := newLargeCache(context.Background(), c, &clock)
	assertEqual(t, err, nil)

	onRemoveDeleted, onRemoveExpired = false, false
	clock.set(5)
	cache.cleanUp(uint64(clock.Epoch()))

	err = cache.Delete("key")
	assertEqual(t, err, ErrEntryNotFound)
	assertEqual(t, false, onRemoveDeleted)
	assertEqual(t, true, onRemoveExpired)

	onRemoveDeleted, onRemoveExpired = false, false
	clock.set(0)

	cache.Set("key2", []byte("value2"))
	err = cache.Delete("key2")
	clock.set(5)
	cache.cleanUp(uint64(clock.Epoch()))

	assertEqual(t, err, nil)
	assertEqual(t, true, onRemoveDeleted)
	assertEqual(t, false, onRemoveExpired)
}

func TestOnRemoveGetEntryStats(t *testing.T) {
	t.Parallel()

	clock := mockedClock{value: 0}
	count := uint32(0)
	onRemove := func(key string, entry []byte, keyMetada Metadata) {
		count = keyMetada.RequestCount
	}
	c := Config{
		Shards:               1,
		LifeWindow:           time.Second,
		MaxEntriesInWindow:   1,
		MaxEntriesSize:       256,
		OnRemoveWithMetadata: onRemove,
		StatsEnabled:         true,
	}.OnRemoveFilterSet(Deleted, NoSpace)

	cache, _ := newLargeCache(context.Background(), c, &clock)

	cache.Set("key", []byte("value"))

	for i := 0; i < 100; i++ {
		cache.Get("key")
	}

	cache.Delete("key")

	assertEqual(t, uint32(100), count)
}

func TestCacheLen(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})
	keys := 1337

	for i := 0; i < keys; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	assertEqual(t, keys, cache.Len())
}

func TestCacheCapacity(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})
	keys := 1337

	for i := 0; i < keys; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	assertEqual(t, keys, cache.Len())
	assertEqual(t, 40960, cache.Capacity())
}

func TestCacheInitialCapacity(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 2 * 1024,
		HardMaxCacheSize:   1,
		MaxEntriesSize:     1024,
	})

	assertEqual(t, 0, cache.Len())
	assertEqual(t, 1024*1024, cache.Capacity())

	keys := 1024 * 1024
	for i := 0; i < keys; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	assertEqual(t, true, cache.Len() < keys)
	assertEqual(t, 1024*1024, cache.Capacity())
}

func TestRemoveEntriesWhenShardIsFull(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             1,
		LifeWindow:         100 * time.Second,
		MaxEntriesInWindow: 100,
		MaxEntriesSize:     256,
		HardMaxCacheSize:   1,
	})

	value := blob('a', 1024*300)

	cache.Set("key", value)
	cache.Set("key", value)
	cache.Set("key", value)
	cache.Set("key", value)
	cache.Set("key", value)
	cacheValue, err := cache.Get("key")

	noError(t, err)
	assertEqual(t, value, cacheValue)
}

func TestCacheStats(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})

	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	for i := 0; i < 10; i++ {
		value, err := cache.Get(fmt.Sprintf("key%d", i))
		noError(t, err)
		assertEqual(t, string(value), "value")
	}

	for i := 100; i < 110; i++ {
		_, err := cache.Get(fmt.Sprintf("key%d", i))
		assertEqual(t, ErrEntryNotFound, err)
	}

	for i := 10; i < 20; i++ {
		err := cache.Delete(fmt.Sprintf("key%d", i))
		assertEqual(t, ErrEntryNotFound, err)
	}

	for i := 110; i < 120; i++ {
		err := cache.Delete(fmt.Sprintf("key%d", i))
		assertEqual(t, ErrEntryNotFound, err)
	}

	stats := cache.Stats()
	assertEqual(t, stats.Hits, int64(10))
	assertEqual(t, stats.Misses, int64(10))
	assertEqual(t, stats.DelHits, int64(10))
	assertEqual(t, stats.DelMissed, int64(10))
}

func TestCacheEntryStats(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             10,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
		StatsEnabled:       true,
	})

	cache.Set("key0", []byte("value"))

	for i := 0; i < 10; i++ {
		_, err := cache.Get("key0")
		noError(t, err)
	}

	keyMetadata := cache.keyMetadata("key0")
	assertEqual(t, uint32(10), keyMetadata.RequestCount)
}

func TestCacheEntryStat(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
		StatsEnabled:       true,
	})

	cache.Set("key0", []byte("value"))
	for i := 0; i < 0; i++ {
		_, err := cache.Get("key0")
		noError(t, err)
	}

	keyMetadata := cache.keyMetadata("key0")
	assertEqual(t, uint32(10), keyMetadata.RequestCount)
}

func TestCacheResetStats(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})

	for i := 0; i < 10; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	for i := 0; i < 10; i++ {
		value, err := cache.Get(fmt.Sprintf("key%d", i))
		noError(t, err)
		assertEqual(t, string(value), "value")
	}
	for i := 100; i < 110; i++ {
		_, err := cache.Get(fmt.Sprintf("key%d", i))
		noError(t, err)
		assertEqual(t, ErrEntryNotFound, err)
	}

	for i := 10; i < 20; i++ {
		err := cache.Delete(fmt.Sprintf("key%d", i))
		noError(t, err)
	}

	for i := 110; i < 120; i++ {
		err := cache.Delete(fmt.Sprintf("key%d", i))
		assertEqual(t, ErrEntryNotFound, err)
	}

	stats := cache.Stats()
	assertEqual(t, stats.Hits, int64(10))
	assertEqual(t, stats.Misses, int64(10))
	assertEqual(t, stats.DelHits, int64(10))
	assertEqual(t, stats.DelMissed, int64(10))

	cache.ResetStats()
	stats = cache.Stats()
	assertEqual(t, stats.Hits, int64(0))
	assertEqual(t, stats.Misses, int64(0))
	assertEqual(t, stats.DelHits, int64(0))
	assertEqual(t, stats.DelMissed, int64(0))
}

func TestCacheDel(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), DefaultConf(time.Second))

	err := cache.Delete("nonExistingKey")

	assertEqual(t, err, ErrEntryNotFound)

	cache.Set("existingKey", nil)
	err = cache.Delete("existingKey")
	cacheValue, _ := cache.Get("existingKey")

	noError(t, err)
	assertEqual(t, 0, len(cacheValue))
}

func TestCacheDelRandomly(t *testing.T) {
	t.Parallel()

	c := Config{
		Shards:             1,
		LifeWindow:         time.Second,
		CleanWindow:        0,
		MaxEntriesInWindow: 10,
		MaxEntriesSize:     10,
		Verbose:            false,
		Hasher:             newDefaultHasher(),
		HardMaxCacheSize:   1,
		StatsEnabled:       true,
		Logger:             DefaultLogger(),
	}

	cache, _ := New(context.Background(), c)
	var wg sync.WaitGroup
	var ntest = 800000
	wg.Add(3)
	go func() {
		for i := 0; i < ntest; i++ {
			r := uint8(rand.Int())
			key := fmt.Sprintln("thekey%d", r)
			cache.Delete(key)
		}
		wg.Done()
	}()

	valueLen := 1024
	go func() {
		val := make([]byte, valueLen)
		for i := 0; i < ntest; i++ {
			r := byte(rand.Int())
			key := fmt.Sprintf("thekey%d", r)

			for j := 0; j < len(val); j++ {
				val[j] = r
			}
			cache.Set(key, val)
		}
		wg.Done()
	}()
	go func() {
		val := make([]byte, valueLen)
		for i := 0; i < ntest; i++ {
			r := byte(rand.Int())
			key := fmt.Sprintf("thekey%d", r)

			for j := 0; j < len(val); j++ {
				val[j] = r
			}

			if got, err := cache.Get(key); err == nil && !bytes.Equal(got, val) {
				t.Errorf("got %s -> \n %x\n expected:\n %x", key, got, val)
			}
		}
		wg.Done()
	}()
	wg.Wait()
}

func TestWriteAndReadParallelSameKeyWithStats(t *testing.T) {
	t.Parallel()

	c := DefaultConf(10 * time.Second)
	c.StatsEnabled = true

	cache, _ := New(context.Background(), c)
	var wg sync.WaitGroup
	ntest := 1000
	n := 10
	wg.Add(n)
	key := "key"
	value := blob('a', 1024)
	for i := 0; i < ntest; i++ {
		assertEqual(t, nil, cache.Set(key, value))
	}

	for j := 0; j < n; j++ {
		go func() {
			for i := 0; i < ntest; i++ {
				v, err := cache.Get(key)
				assertEqual(t, nil, err)
				assertEqual(t, value, v)
			}
		}()
		wg.Done()
	}

	wg.Wait()

	assertEqual(t, Stats{Hits: int64(n * ntest)}, cache.Stats())
	assertEqual(t, ntest*n, int(cache.keyMetadata(key).RequestCount))
}

func TestCacheReset(t *testing.T) {
	t.Parallel()

	cache, _ := New(context.Background(), Config{
		Shards:             8,
		LifeWindow:         time.Second,
		MaxEntriesInWindow: 1,
		MaxEntriesSize:     256,
	})
	keys := 1337

	for i := 0; i < keys; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	assertEqual(t, keys, cache.Len())

	cache.Reset()

	assertEqual(t, 0, cache.Len())

	for i := 0; i < keys; i++ {
		cache.Set(fmt.Sprintf("key%d", i), []byte("value"))
	}

	assertEqual(t, keys, cache.Len())
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
