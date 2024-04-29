package largecache

import "unsafe"

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
