package leaking_bucket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fedragon/rate-limiter/common"
	"github.com/fedragon/rate-limiter/test"
	"github.com/stretchr/testify/assert"
)

const (
	route = "/bar"
)

func Test_ServerReturns200_WhenWithinLimits(t *testing.T) {
	rl := NewRateLimiter(&common.Rate{Value: 1, Interval: time.Second})
	defer rl.Stop()
	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 500 * time.Millisecond}
	statusCode, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
}

func Test_ServerReturns429_OnTooManyRequests(t *testing.T) {
	rl := NewRateLimiter(&common.Rate{Value: 1, Interval: time.Second})
	defer rl.Stop()
	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 500 * time.Millisecond}

	statusCode, err := sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)

	statusCode, err = sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, statusCode)
}

func Test_ServerReturns200_AfterRefill(t *testing.T) {
	rl := NewRateLimiter(&common.Rate{Value: 1, Interval: time.Second})
	defer rl.Stop()
	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 500 * time.Millisecond}

	statusCode, err := sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)

	statusCode, err = sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, statusCode)

	time.Sleep(time.Second)

	statusCode, err = sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
}

func sendRequest(route string, client *http.Client) (int, error) {
	req, _ := http.NewRequest("GET", route, nil)

	res, err := client.Do(req)
	defer res.Body.Close()

	return res.StatusCode, err
}
