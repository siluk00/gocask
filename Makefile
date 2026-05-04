.PHONY: test bench lint build

test:
	go test ./... -race -v

bench:
	go test ./... -bench=. -benchmem -run='^$$'

lint:
	golangci-lint run ./...

build:
	go build -o bin/gocask ./cmd/gocask
