package largecache

import "sync"

type iteratorError string

func (e iteratorError) Error() string {
	return string(e)
}

const ErrInvalidIteratorState = iteratorError("Iterator is in invalid state. Use SetNext() to move to next position")

const ErrCannotRetrieveEntry = iteratorError("Could not retrieve entry from cache")

var emptyEntryInfo = EntryInfo{}

type EntryInfo struct {
	timestamp uint64
	hash      uint64
	key       string
	value     []byte
	err       error
}

func (e EntryInfo) Key() string {
	return e.key
}

func (e EntryInfo) Hash() uint64 {
	return e.hash
}

func (e EntryInfo) Timestamp() uint64 {
	return e.timestamp
}

func (e EntryInfo) Value() []byte {
	return e.value
}

type EntryInfoIterator struct {
	mutex sync.Mutex
	//cache *
	currentShard    int
	currentIndex    int
	curentEntryInfo EntryInfo
	elements        []uint64
	elementsCount   int
	valid           bool
}
