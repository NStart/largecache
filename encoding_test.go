package largecache

import (
	"testing"
	"time"
)

func TestEncodeDecode(t *testing.T) {
	now := uint64(time.Now().Unix())
	hash := uint64(42)
	key := "key"
	data := []byte("data")
	buffer := make([]byte, 100)

	wrapped := wrapEntry(now, hash, key, data, &buffer)

	assertEqual(t, key, readKeyFromEntry(wrapped))
	assertEqual(t, hash, readHashFromEntry(wrapped))
	assertEqual(t, now, readTimestampFromEntry(wrapped))
	assertEqual(t, data, readEntry(wrapped))
	assertEqual(t, 100, len(buffer))
}
