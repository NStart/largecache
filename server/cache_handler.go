package main

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func cacheIndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:

		}
	})
}

func cacheClearHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clearCache(w, r)
	})
}

func clearCache(w http.ResponseWriter, r *http.Request) {
	if err := cache.Reset(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("internal cache error: %s", err)
	}
	log.Println("cache is successfully cleared")
	w.WriteHeader(http.StatusOK)
}

func getCacheHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path[len(cachePath):]
	if target == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("can't get a key if there is no key."))
		log.Print("empty request.")
		return
	}
	entry, err := cache.Get(target)
	if err != nil {
		errMsg := (err).Error()
		if strings.Contains(errMsg, "not found") {
			log.Print(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(entry)
}

func putCacheHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path[len(cachePath):]
	if target == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("can't put a key if there is no key"))
		log.Print("empty request.")
		return
	}

	entry, err := io.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := cache.Set(target, []byte(entry)); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("stored \"%s\" in  cache.", target)
	w.WriteHeader(http.StatusCreated)
}

func deleteCacheHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path[len(cachePath):]
	if err := cache.Delete(target); err != nil {
		if strings.Contains((err).Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
			log.Printf("%s not found.", target)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("internal cache error: %s", err)
	}
	w.WriteHeader(http.StatusOK)
}