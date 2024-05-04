package largecache

import (
	"context"
	"errors"
)

const (
	minimumEntriesInShard = 10
)

type LargeCache struct {
	shards     []*cacheShard
	lifeWindow uint64
	clock      clock
	hash       Hasher
	config     Config
	shardMask  uint64
	close      chan struct{}
}

type Response struct {
	EntryStatus RemoveReason
}

type RemoveReason uint32

const (
	Expried = RemoveReason(1)
	NoSpace = RemoveReason(2)
	Deleted = RemoveReason(3)
)

func New(ctx context.Context, config Config) (*LargeCache, error) {
	//return
}

func newLargeCache(ctx context.Context, config Config, clock clock) (*LargeCache, error) {
	if !isPowerOfTwo(config.Shards) {
		return nil, errors.New("Shards number must be power of two")
	}

	if config.MaxEntriesSize < 0 {
		return nil, errors.New("MaxEntrySize must be >= 0")
	}

	if config.HardMaxCacheSize < 0 {
		return nil, errors.New("HardMaxSize must be >= 0")
	}

	lifeWindowSeconds := uint64(config.LifeWindow.Seconds())
	if config.CleanWindow > 0 && lifeWindowSeconds == 0 {
		return nil, errors.New("LifeWindow must be >= 1s when CleanWindow is set")
	}
	if config.Hasher == nil {
		config.Hasher = newDefaultHasher()
	}

	cache := &LargeCache{
		shards:     make([]*cacheShard, config.Shards),
		lifeWindow: lifeWindowSeconds,
		clock:      clock,
		hash:       config.Hasher,
		config:     config,
		shardMask:  uint64(config.Shards - 1),
		close:      make(chan struct{}),
	}
}
