package main

import (
	"bytes"
	"context"
	"io"
	"largecache"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testBaseString = "http://largecache.org"
)

func testCacheSetup() {
	cache, _ = largecache.New(context.Background(), largecache.Config{
		Shards:             1024,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntriesSize:     500,
		Verbose:            true,
		HardMaxCacheSize:   8192,
		OnRemove:           nil,
	})
}

func TestMain(m *testing.M) {
	testCacheSetup()
	m.Run()
}

func TestGetWithNoKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/", nil)
	rr := httptest.NewRecorder()

	getCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 400 {
		t.Errorf("want: 400; got: %d", resp.StatusCode)
	}
}

func TestGetWithMiasingKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/doseNotExist", nil)
	rr := httptest.NewRecorder()

	getCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 404 {
		t.Errorf("want: 404; got: %d", resp.StatusCode)
	}
}

func testGetKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/getKey", nil)
	rr := httptest.NewRecorder()

	cache.Set("getkey", []byte("123"))

	getCacheHandler(rr, req)
	resp := rr.Result()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("cannot deserialise test response: %s", err)
	}

	if string(body) != "123" {
		t.Errorf("want: 123; got: %s.\n\tcan't get existing key getkey.", string(body))
	}
}

func TestPutKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/putkey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	putCacheHandler(rr, req)

	testPutKeyResult, err := cache.Get("putkey")
	if err != nil {
		t.Errorf("error returing cache entry: %s", err)
	}

	if string(testPutKeyResult) != "123" {
		t.Errorf("want: 123; got: %s.\n\tcan't get PUT key putkey", string(testPutKeyResult))
	}
}

func testPutEmptyKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	putCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 400 {
		t.Errorf("want: 400; got: %d.\n\tempty key insertion should return with 400", resp.StatusCode)
	}
}

func TestDeleteEmptyKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	deleteCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 404 {
		t.Errorf("want: 404; got: %d.\n\tapparently we're trying to delete empty keys.", resp.StatusCode)
	}
}

func TestDelteInvalidKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/invalidDeleteKey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	deleteCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 404 {
		t.Errorf("want: 404l; got: %d.\n\tapparently we're trying to delete invalid keys.", resp.StatusCode)
	}
}

func TestDeleteKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cahce/testDeleteKey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	if err := cache.Set("testDeleteKey", []byte("123")); err != nil {
		t.Errorf("can't set key for testing. %s", err)
	}
	deleteCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 200 {
		t.Errorf("want: 200; got: %d.\n\tcan't delete keys", resp.StatusCode)
	}
}

func TestClearCache(t *testing.T) {
	t.Parallel()

	putRequest := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/putkey", bytes.NewBuffer([]byte("123")))
	putResponseRecoder := httptest.NewRecorder()

	putCacheHandler(putResponseRecoder, putRequest)

	requestClear := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/clear", nil)
	rr := httptest.NewRecorder()

	if err := cache.Set("testDeleteKey", []byte("123")); err != nil {
		t.Errorf("can't set key for testing. %s", err)
	}

	clearCache(rr, requestClear)
	resp := rr.Result()

	if resp.StatusCode != 200 {
		t.Errorf("want: 200; got: %d\n\tcan't delete keys", resp.StatusCode)
	}
}

func TestGetStats(t *testing.T) {

}
