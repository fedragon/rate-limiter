package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fedragon/rate-limiter/common"
	"github.com/fedragon/rate-limiter/leaking_bucket"
)

func main() {
	rate := &common.Rate{
		Value:    2,
		Interval: time.Minute,
	}
	rateLimiter := leaking_bucket.NewRateLimiter(rate)
	defer rateLimiter.Stop()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, os.Kill)

	handler := rateLimiter.Handle(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			return
		}),
	)
	server := &http.Server{
		Addr:    ":3000",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
			return
		}
	}()

	<-shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
}
