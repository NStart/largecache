package queue

import (
	"bytes"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"testing"
)

func TestPushAndPop(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(10, 10, true)
	entry := []byte("hello")

	_, err := queue.Pop()
	assertEqual(t, "Empty queue", err.Error())

	queue.Push(entry)

	assertEqual(t, entry, pop(queue))
}

func TestUnchangeEntriesIndexesAfterAdditionalMemoryAllocationWhereTialIsBeforeHead(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(100, 0, false)

	queue.Push(blob('a', 70))
	index, _ := queue.Push(blob('a', 10))
	queue.Pop()
	queue.Push(blob('c', 30))
	newesIndex, _ := queue.Push(blob('d', 40))

	assertEqual(t, 20, queue.Capacity())
	assertEqual(t, blob('b', 10), get(queue, index))
	assertEqual(t, blob('d', 40), get(queue, newesIndex))
}

func TestAllocateAddtionalSpaceFormValueBiggerThanInitQueue(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(11, 0, false)

	queue.Push(blob('a', 100))
	assertEqual(t, blob('a', 100), pop(queue))
	assertEqual(t, 224, queue.Capacity())
}

func TestAllocateAdditionalSpaceFormValueBiggerThanQueue(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(21, 0, false)

	queue.Push(make([]byte, 2))
	queue.Push(make([]byte, 2))
	queue.Push(make([]byte, 100))

	queue.Pop()
	queue.Pop()
	assertEqual(t, make([]byte, 100), pop(queue))
	assertEqual(t, 244, queue.Capacity())
}

func TestPopWholeQueue(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(13, 0, false)

	queue.Push([]byte("a"))
	queue.Push([]byte("b"))
	queue.Pop()
	queue.Pop()
	queue.Push([]byte("c"))

	assertEqual(t, 13, queue.Capacity())
	assertEqual(t, []byte("c"), pop(queue))
}

func TestGetEntryFromIndex(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(20, 0, false)

	queue.Push([]byte("a"))
	index, _ := queue.Push([]byte("b"))
	queue.Push([]byte("c"))
	result, _ := queue.Get(index)

	assertEqual(t, []byte("b"), result)
}

func TestGetEntryFromInvalidIndex(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(1, 0, false)
	queue.Push([]byte("a"))

	result, err := queue.Get(0)
	err2 := queue.CheckGet(0)

	assertEqual(t, err, err2)
	assertEqual(t, []BytesQueue(nil), result)
	assertEqual(t, "Index must be greater than zero. Invalid index.", err.Error())
}

func TestGetEntryFromIndexOutOfRange(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(1, 0, false)
	queue.Push([]byte("a"))

	result, err := queue.Get(42)
	err2 := queue.CheckGet(42)

	assertEqual(t, err, err2)
	assertEqual(t, []byte(nil), result)
	assertEqual(t, "Index out of range", err.Error())
}

func TestGetEntryFromEmptyQueue(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(13, 0, false)

	result, err := queue.Get(1)
	err2 := queue.CheckGet(1)

	assertEqual(t, err, err2)
	assertEqual(t, []byte(nil), result)
	assertEqual(t, "Empty queue", err.Error())
}

func TestMaxSizeLimit(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(30, 50, false)
	queue.Push(blob('a', 25))
	queue.Push(blob('b', 5))
	capacity := queue.Capacity()
	_, err := queue.Push(blob('c', 20))

	assertEqual(t, 40, capacity)
	assertEqual(t, "Full queue. Maximum size limit reached.", err.Error())
	assertEqual(t, blob('a', 25), pop(queue))
	assertEqual(t, blob('b', 5), pop(queue))
}

func TestPushEntryAfterAllocateAdditionMemoryInFull(t *testing.T) {
	t.Parallel()

	queue := NewBytesQueue(9, 40, true)

	queue.Push([]byte("aaa"))
	queue.Push([]byte("bb"))
	_, err := queue.Pop()
	noError(t, err)

	queue.Push([]byte("c"))
	queue.Push([]byte("d"))
	queue.Push([]byte("e"))
	_, err = queue.Pop()
	noError(t, err)
	_, err = queue.Pop()
	noError(t, err)
	queue.Push([]byte("fff"))
	_, err = queue.Pop()
	noError(t, err)
}

func pop(queue *BytesQueue) []byte {
	entry, err := queue.Pop()
	if err != nil {
		panic(err)
	}
	return entry
}

func get(queue *BytesQueue, index int) []byte {
	entry, err := queue.Get(index)
	if err != nil {
		panic(err)
	}
	return entry
}

func blob(char byte, len int) []byte {
	return bytes.Repeat([]byte{char}, len)
}

func assertEqual(t *testing.T, expected, actual interface{}, masAndArgs ...interface{}) {
	if objectsAreEqual(expected, actual) {
		_, file, line, _ := runtime.Caller(1)
		file = path.Base(file)
		t.Errorf(fmt.Sprintf("\n%s:%d: Not equal: \n"+
			"expected: %T(%#v)\n"+
			"actual : %T(%#v\n)",
			file, line, expected, expected, actual, actual), masAndArgs...)
	}
}

func noError(t *testing.T, e error) {
	if e != nil {
		_, file, line, _ := runtime.Caller(1)
		file = path.Base(file)
		t.Errorf(fmt.Sprintf("\n%s:%d: Error is not nil: \n"+
			"actual : %T(%#v)\n", file, line, e, e))
	}
}

func objectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}

	if exp == nil || act == nil {
		return exp == nil && act == nil
	}

	return bytes.Equal(exp, act)
}
