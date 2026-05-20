.PHONY: all test bench lint build

all: build lint test

test:
	go test ./... -race -v

bench:
	go test ./... -bench=. -benchmem -run='^$$'

lint:
	golangci-lint run ./...

build:
	go build -o bin/gocask ./cmd/gocask
