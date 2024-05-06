package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	t.Parallel()
	var testStats largecache.Stats

	req := httptest.NewRequest("GET", testBaseString+"/api/v1/stats", nil)
	rr := httptest.NewRecorder()

	if err := cache.Set("increamentStats", []byte("123")); err != nil {
		t.Errorf("error settting cache value. error: %s", err)
	}

	if _, err := cache.Get("incrementStats"); err != nil {
		t.Errorf("can't find incrementStats. error: %s", err)
	}

	getCacheHandler(rr, req)
	resp := rr.Result()

	if err := json.NewDecoder(resp.Body).Decode(&testStats); err != nil {
		t.Errorf("error decoding cache stats. error: %s", err)
	}

	if testStats.Hits == 0 {
		t.Errorf("want: > 0; got: 0.\n\thandler not properly returning stats info.")
	}
}

func TestStatsIndex(t *testing.T) {
	t.Parallel()
	var testStats largecache.Stats

	getreq := httptest.NewRequest("GET", testBaseString+"/api/v1/stats", nil)
	putreq := httptest.NewRequest("PUT", testBaseString+"/api/v1/stats", nil)
	rr := httptest.NewRecorder()

	if err := cache.Set("incrementStats", []byte("123")); err != nil {
		t.Errorf("error setting cache value. error: %s", err)
	}

	if _, err := cache.Get("incrementStats"); err != nil {
		t.Errorf("can't find incrementStats. error: %s", err)
	}

	testHandlers := statsIndexHandler()
	testHandlers.ServeHTTP(rr, getreq)
	resp := rr.Result()

	if err := json.NewDecoder(resp.Body).Decode(&testStats); err != nil {
		t.Errorf("error decoding cache stats. error: %s", err)
	}

	if testStats.Hits == 0 {
		t.Errorf("want: > 0.\n\thandler not properly returning stats info.")
	}

	testHandlers = statsIndexHandler()
	testHandlers.ServeHTTP(rr, putreq)
	resp = rr.Result()
	_, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("cannot deserialise test response: %s", err)
	}
}

func TestCacheIndexHandler(t *testing.T) {
	getreq := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/testkey", nil)
	putreq := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/testkey", bytes.NewBuffer([]byte("123")))
	delreq := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/testkey", bytes.NewBuffer([]byte("123")))

	getrr := httptest.NewRecorder()
	putrr := httptest.NewRecorder()
	delrr := httptest.NewRecorder()
	testHandler := cacheIndexHandler()

	testHandler.ServeHTTP(putrr, putreq)
	resp := putrr.Result()
	if resp.StatusCode != 201 {
		t.Errorf("want: 201; got: %d.\n\tcan't put keys", resp.StatusCode)
	}
	testHandler.ServeHTTP(getrr, getreq)
	resp = getrr.Result()
	if resp.StatusCode != 200 {
		t.Errorf("want: 200; got: %d.\n\tcan't get keys.", resp.StatusCode)
	}

	testHandler.ServeHTTP(delrr, delreq)
	resp = delrr.Result()
	if resp.StatusCode != 200 {
		t.Errorf("want: 200; got: %d.\n\tcan't delete keys.", resp.StatusCode)
	}
}

func TestInvalidPutWhenExceedShardCap(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/putkey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	putCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 500 {
		t.Errorf("want: 500; got: %d", resp.StatusCode)
	}
}

func TestValidPutWhenReading(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/putkey", errReader(0))
	rr := httptest.NewRecorder()

	putCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 500 {
		t.Errorf("want: 500; got: %d", resp.StatusCode)
	}
}

type errReader int

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("test read error")
}
