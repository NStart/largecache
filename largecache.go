package largecache

const (
	minimumEntriesInShard = 10
)

type Response struct {
	EntryStatus RemoveReason
}

type RemoveReason uint32
