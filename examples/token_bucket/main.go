package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fedragon/rate-limiter/common"
	"github.com/fedragon/rate-limiter/token_bucket"
)

func main() {
	rate := common.Rate{
		Value:    2,
		Interval: time.Minute,
	}
	rateLimiter, err := token_bucket.
		NewRateLimiterBuilder().
		RegisterUser("0-0-0-0-0").
		SetLimit(
			"/",
			token_bucket.Config{
				Limit:  rate,
				Refill: rate,
			}).
		Build()
	if err != nil {
		log.Fatal(err)
	}
	defer rateLimiter.Stop()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, os.Kill)

	handler := rateLimiter.Handle(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			return
		}),
	)

	go func() {
		if err := http.ListenAndServe("0.0.0.0:3000", handler); err != nil {
			log.Fatal(err)
			return
		}
	}()

	<-shutdown
}
