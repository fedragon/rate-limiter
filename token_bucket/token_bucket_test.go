package token_bucket

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
	route  = "/bar"
	userID = "0-0-0-0-0"
)

var (
	limit = Limit{
		Limit: common.Rate{
			Value:    1,
			Interval: time.Second,
		},
		Refill: common.Rate{
			Value:    1,
			Interval: 2 * time.Second,
		},
	}
)

func TestRateLimiterBuilder_Build_FailsIfLimitsAreNotConfigured(t *testing.T) {
	rlb := NewRateLimiterBuilder()
	rl, err := rlb.Build()

	assert.Nil(t, rl)
	assert.Error(t, err)
}

func Test_ServerReturns401_IfUserIsUnknown(t *testing.T) {
	rlb := NewRateLimiterBuilder()
	rl, _ := rlb.SetLimit(route, limit).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	statusCode, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, statusCode)
}

func Test_ServerReturns200_WhenWithinLimits(t *testing.T) {
	rlb := NewRateLimiterBuilder()
	rl, _ := rlb.SetLimit(route, limit).RegisterUser(userID).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	statusCode, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
}

func Test_ServerReturns200_AfterRefill(t *testing.T) {
	rlb := NewRateLimiterBuilder()
	rl, _ := rlb.SetLimit(route, limit).RegisterUser(userID).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 5 * time.Second}
	statusCode, err := sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)

	time.Sleep(3 * time.Second)

	statusCode, err = sendRequest(server.URL+route, client)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
}

func Test_ServerReturns429_OnTooManyRequests(t *testing.T) {
	rlb := NewRateLimiterBuilder()
	rl, _ := rlb.SetLimit(route, limit).RegisterUser(userID).Build()
	defer rl.Stop()

	server := httptest.NewServer(rl.Handle(test.ItsOK()))

	client := &http.Client{Timeout: 5 * time.Second}

	statusCode, err := sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)

	statusCode, err = sendRequest(server.URL+route, client)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, statusCode)
}

func sendRequest(route string, client *http.Client) (int, error) {
	req, _ := http.NewRequest("GET", route, nil)
	req.Header.Set("X-User-ID", userID)

	res, err := client.Do(req)
	defer res.Body.Close()
	return res.StatusCode, err
}
