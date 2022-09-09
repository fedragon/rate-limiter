package rate_limiter

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	route = "/bar"
)

func Test_ServerReturns200_WhenWithinLimits(t *testing.T) {
	limit := 1
	server := httptest.NewServer(RateLimit(route, limit, http.NotFoundHandler()))

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func Test_ServerReturns429_OnTooManyRequests(t *testing.T) {
	server := httptest.NewServer(RateLimit(route, 1, http.NotFoundHandler()))

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
	req.Header.Set("X-User-ID", "123")

	return client.Do(req)
}
