BINARY  := capsule
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/capsule

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	go clean -testcache
