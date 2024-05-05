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

	var onRemove func(wrappedEntry []byte, reason RemoveReason)
	if config.OnRemoveWithMetadata != nil {
		onRemove = cache.provideOnRemoveWithMetadata
	} else if config.OnRemove != nil {
		onRemove = cache.providedOnRemove
	} else if config.OnRemoveWithReason != nil {
		//onRemove = cache.Pro
	}
}

func (c *LargeCache) Close() error {
	close(c.close)
	return nil
}

func (c *LargeCache) Get(key string) ([]byte, error) {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	return shard.get(key, hashedKey)
}

func (c *LargeCache) GetWithInfo(key string) ([]byte, Response, error) {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	return shard.getWithInfo(key, hashedKey)
}

func (c *LargeCache) Set(key string, entry []byte) error {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	return shard.set(key, hashedKey, entry)
}

func (c *LargeCache) Append(key string, entry []byte) error {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	return shard.append(key, hashedKey, entry)
}

func (c *LargeCache) Delete(key string) error {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	return shard.del(hashedKey)
}

func (c *LargeCache) Reset() error {
	for _, shard := range c.shards {
		shard.reset(c.config)
	}
	return nil
}

func (c *LargeCache) ResetStats() error {
	for _, shard := range c.shards {
		shard.resetStats()
	}
	return nil
}

func (c *LargeCache) Len() int {
	var len int
	for _, shard := range c.shards {
		len += shard.len()
	}
	return len
}

func (c *LargeCache) Capacity() int {
	var len int
	for _, shard := range c.shards {
		len += shard.capacity()
	}
	return len
}

func (c *LargeCache) Stats() Stats {
	var s Stats
	for _, shard := range c.shards {
		tmp := shard.GetStats()
		s.Hits += tmp.Hits
		s.Misses += tmp.Misses
		s.DelHits += tmp.DelHits
		s.DelMissed += tmp.DelMissed
		s.Collision += tmp.Collision
	}
	return s
}

func (c *LargeCache) keyMetadata(key string) Metadata {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	return shard.getKeyMetadataWithLock(hashedKey)
}

func (c *LargeCache) Interator() *EntryInfoIterator {
	return newInterator(c)
}

func (c *LargeCache) onEvict(oldestEntry []byte, currentTimestamp uint64, evict func(reason RemoveReason) error) bool {
	oldestTimestamp := readTimestampFromEntry(oldestEntry)
	if currentTimestamp < oldestTimestamp {
		return false
	}

	if currentTimestamp-oldestTimestamp > c.lifeWindow {
		evict(Expried)
		return true
	}
	return false
}

func (c *LargeCache) cleanUp(currentTimestamp uint64) {
	for _, shard := range c.shards {
		shard.cleanUp(currentTimestamp)
	}
}

func (c *LargeCache) getShard(hashKey uint64) (shard *cacheShard) {
	return c.shards[hashKey&c.shardMask]
}

func (c *LargeCache) providedOnRemove(wrappedEntry []byte, reason RemoveReason) {
	c.config.OnRemove(readKeyFromEntry(wrappedEntry), readEntry(wrappedEntry))
}

func (c *LargeCache) providedOnRemoveWithReason(wrappedEntry []byte, reason RemoveReason) {
	if c.config.onRemoveFilter == 0 || (1<<uint(reason))&c.config.onRemoveFilter > 0 {
		c.config.OnRemoveWithReason(readKeyFromEntry(wrappedEntry), readEntry(wrappedEntry), reason)
	}
}

func (c *LargeCache) notProvidedOnRemove(wrappedEntry []byte, reason RemoveReason) {

}

func (c *LargeCache) providedOnRemoveWithMetadata(wrappedEntry []byte, reason RemoveReason) {
	key := readKeyFromEntry()

	hashKey := c.hash.Sum64(key)
	shards := c.getShard(hashKey)
	c.config.OnRemoveWithMetadata(key, readEntry(wrappedEntry), shards.)
}