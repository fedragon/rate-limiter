package test

import (
	"math/rand"
	"net/http"
	"time"
)

// ItsOK returns an HTTP handler for testing purposes, that always responds with 200 OK.
func ItsOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	})
}

// RandomDuration returns a random duration between 0 and 50ms.
func RandomDuration() time.Duration {
	return time.Duration(rand.Int31n(51)) * time.Millisecond
}
