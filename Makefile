.DEFAULT_GOAL := all
.PHONY: test

all: test

test:
	go test -short -race -count=1 -v ./...