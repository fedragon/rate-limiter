package test

import "net/http"

// ItsOK returns an HTTP handler for testing purposes, that always responds with 200 OK.
func ItsOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	})
}
