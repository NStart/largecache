package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func statsIndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCacheStatsHandler(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func getCacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	target, err := json.Marshal(cache.Stats())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("cannot marshal cache stats. eror: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(target)
}
