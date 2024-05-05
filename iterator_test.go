package largecache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestEntriesIntera(t *testing.T) {
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

func TestEntriesInteratorWithMostShardsEmpty(t *testing.T) {
	t.Parallel()

	clock := m
}
