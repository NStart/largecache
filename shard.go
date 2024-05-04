package largecache

type onRemoveCallBack func(wrappedEntry []byte, reason RemoveReason)

type Metadata struct {
	RequestCount uint32
}

type cacheShard struct {
	hashmap map[uint64]uint64
}
