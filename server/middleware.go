package main

import (
	"log"
	"net/http"
	"time"
)

type service func(http.Handler) http.Handler

func serviceLoader(h http.Handler, svcs ...service) http.Handler {
	for _, svc := range svcs {
		h = svc(h)
	}
	return h
}

func requestMetrics(l *log.Logger) service {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			h.ServeHTTP(w, r)
			l.Printf("%s request to %d took %vns.", r.Method, r.URL.Path, time.Since(start).Nanoseconds())
		})
	}
}
