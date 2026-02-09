BINARY  := capsule
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test test-full smoke lint clean hooks

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/capsule

test:
	go test -short ./...

test-full:
	go test ./...

smoke:
	go test -tags smoke ./cmd/capsule/

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	go clean -testcache

hooks:
	cp scripts/hooks/pre-commit.sh .git/hooks/pre-commit.old
	chmod +x .git/hooks/pre-commit.old
