package largecache

import "time"

// Config for LargeCache
type Config struct {
	// Number of cache shards, value must be a power of two
	shards int
	// Time after which entry can be evicted
	LifeWindow time.Duration
	// Interval between removing expired entries (clean up).
	// If set to <= 0 then no action is performed. Setting to < 1 second is counterproductive - largecache has a one second resolution.
	CleanWindow time.Duration
	// Max number of entries in life window. Used only to calculate initial size for cache shards.
	// When proper value is set then addtional memory allocation does not occur.
	MaxEntriesInWindow   int
	MaxEntriesSize int
	StatsEnabled          bool
	Verbose bool
	Hasher               Hasher
	HardMaxCacheSize     int
	OnRemove             func(key string, entry []byte)
	OnRemoveWithMetadata func(key string, entry []byte, keyMetadata Metadata)
	OnRemoveWithReason   func(key string, entry []byte, reason RemoveReason)

	onRemoveFilter int

	Logger Logger
}

func DefaultConf(eviction time.Duration) Config {
	return Config{
		shards: 1024,
		LifeWindow: eviction,
		CleanWindow: 1 * time.Second,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntriesSize: 500,
		StatsEnabled: false,
		Ver
	}
}
