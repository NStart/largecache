package largecache

import (
	"sync"
)

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
	mutex           sync.Mutex
	cache           *LargeCache
	currentShard    int
	currentIndex    int
	curentEntryInfo EntryInfo
	elements        []uint64
	elementsCount   int
	valid           bool
}

func (it *EntryInfoIterator) SetNext() bool {
	it.mutex.Lock()

	it.valid = false
	it.currentIndex++

	if it.elementsCount > it.currentIndex {
		it.valid = true

		empty := it.setCurrentEntry()
		it.mutex.Unlock()

		if empty {
			return it.SetNext()
		}
		return true
	}

	for i := it.currentShard + 1; i < it.cache.config.Shards; i++ {
		it.elements, it.elementsCount = it.cache.shards[i].copyHashedKeys()

		if it.elementsCount > 0 {
			it.currentIndex = 0
			it.currentShard = i
			it.valid = true

			empty := it.setCurrentEntry()
			it.mutex.Unlock()

			if empty {
				return it.SetNext()
			}
			return true
		}
	}
	it.mutex.Unlock()
	return false
}

func (it *EntryInfoIterator) setCurrentEntry() bool {
	var entryNotFound = false
	entry, err := it.cache.shards[it.currentShard].getEntry(it.elements[it.currentIndex])

	if err == ErrEntryNotFound {
		it.curentEntryInfo = emptyEntryInfo
		entryNotFound = true
	} else if err != nil {
		it.curentEntryInfo = EntryInfo{
			err: err,
		}
	} else {
		it.curentEntryInfo = EntryInfo{
			timestamp: readHashFromEntry(entry),
			hash:      readHashFromEntry(entry),
			key:       readKeyFromEntry(entry),
			value:     readEntry(entry),
			err:       err,
		}
	}

	return entryNotFound
}

func newInterator(cache *LargeCache) *EntryInfoIterator {
	elements, count := cache.shards[0].copyHashedKeys()

	return &EntryInfoIterator{
		cache:         cache,
		currentShard:  0,
		currentIndex:  -1,
		elements:      elements,
		elementsCount: count,
	}
}

func (it *EntryInfoIterator) value() (EntryInfo, error) {
	if !it.valid {
		return emptyEntryInfo, ErrInvalidIteratorState
	}
	return it.curentEntryInfo, it.curentEntryInfo.err
}
