package rate_limiter

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	route  = "/bar"
	userID = "0-0-0-0-0"
)

func Test_ServerReturns401_IfUserIsUnknown(t *testing.T) {
	limit := 1
	rl := RateLimiter{}
	rl.SetLimit(route, limit)

	server := httptest.NewServer(rl.RateLimit(time.Second, itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func Test_ServerReturns200_WhenWithinLimits(t *testing.T) {
	limit := 1
	rl := RateLimiter{}
	rl.SetLimit(route, limit).RegisterUser(userID)

	server := httptest.NewServer(rl.RateLimit(time.Second, itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func Test_ServerReturns200_AfterRefill(t *testing.T) {
	limit := 1
	rl := RateLimiter{}
	rl.SetLimit(route, limit).RegisterUser(userID)

	server := httptest.NewServer(rl.RateLimit(time.Second, itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	time.Sleep(2 * time.Second)

	res, err = sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func Test_ServerReturns429_OnTooManyRequests(t *testing.T) {
	limit := 1
	rl := RateLimiter{}
	rl.SetLimit(route, limit).RegisterUser(userID)

	server := httptest.NewServer(rl.RateLimit(time.Second, itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}

	res, err := sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	res, err = sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, res.StatusCode)
}

func sendRequest(route string, client *http.Client) (*http.Response, error) {
	req, _ := http.NewRequest("GET", route, nil)
	req.Header.Set("X-User-ID", userID)

	return client.Do(req)
}

func itsOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	})
}
