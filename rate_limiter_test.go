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

var (
	limit = Limit{
		Limit: Rate{
			Value:    1,
			Interval: time.Second,
		},
		Refill: Rate{
			Value:    1,
			Interval: 2 * time.Second,
		},
	}
)

func Test_ServerReturns401_IfUserIsUnknown(t *testing.T) {
	rlb := RateLimiterBuilder{}
	rl := rlb.SetLimit(route, limit).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.RateLimit(itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func Test_ServerReturns200_WhenWithinLimits(t *testing.T) {
	rlb := RateLimiterBuilder{}
	rl := rlb.SetLimit(route, limit).RegisterUser(userID).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.RateLimit(itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func Test_ServerReturns200_AfterRefill(t *testing.T) {
	rlb := RateLimiterBuilder{}
	rl := rlb.SetLimit(route, limit).RegisterUser(userID).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.RateLimit(itsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	time.Sleep(3 * time.Second)

	res, err = sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func Test_ServerReturns429_OnTooManyRequests(t *testing.T) {
	rlb := RateLimiterBuilder{}
	rl := rlb.SetLimit(route, limit).RegisterUser(userID).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.RateLimit(itsOK()))

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
