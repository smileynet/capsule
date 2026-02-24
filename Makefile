BINARY  := capsule
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test test-full test-scripts smoke lint clean hooks fmt gate-feature gate-epic demo demo-clean dev-setup

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/capsule

test:
	go test -short ./...

test-full:
	go test ./...
	@$(MAKE) test-scripts

test-scripts:
	@failed=0; \
	for script in tests/scripts/test-*.sh; do \
		echo "--- $$(basename $$script) ---"; \
		bash "$$script" || failed=1; \
	done; \
	[ $$failed -eq 0 ]

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

fmt:
	goimports -w $$(find . -name '*.go' -not -path './templates/*' -not -path './vendor/*')

gate-feature: lint test-full

gate-epic: lint test-full smoke

TEMPLATE ?= campaign
TARGET   ?=

demo:
	@if [ -n "$(TARGET)" ]; then \
		DIR=$$(bash scripts/setup-template.sh --template=demo-$(TEMPLATE) "$(TARGET)"); \
	else \
		DIR=$$(bash scripts/setup-template.sh --template=demo-$(TEMPLATE)); \
	fi; \
	echo ""; \
	echo "Demo playground created at: $$DIR"; \
	echo ""; \
	echo "Next steps:"; \
	echo "  cd $$DIR"; \
	echo "  bd ready"; \
	echo "  bd list"

demo-clean:
	@if [ -z "$(TARGET)" ]; then \
		echo "ERROR: TARGET is required. Usage: make demo-clean TARGET=/path/to/demo" >&2; \
		exit 1; \
	fi; \
	if [ ! -d "$(TARGET)/.git" ]; then \
		echo "ERROR: $(TARGET) does not look like a demo playground (no .git)" >&2; \
		exit 1; \
	fi; \
	rm -rf "$(TARGET)"; \
	echo "Removed demo playground: $(TARGET)"

dev-setup:
	@echo "=== Checking required tools ==="; \
	ok=true; \
	for cmd in go golangci-lint goimports bd; do \
		if command -v $$cmd >/dev/null 2>&1; then \
			echo "  OK: $$cmd"; \
		else \
			echo "  MISSING: $$cmd"; \
			ok=false; \
		fi; \
	done; \
	if [ "$$ok" = false ]; then \
		echo ""; \
		echo "Install missing tools before continuing." >&2; \
		exit 1; \
	fi; \
	echo ""; \
	echo "=== Installing hooks ==="; \
	$(MAKE) hooks; \
	echo ""; \
	echo "=== Building ==="; \
	$(MAKE) build; \
	echo ""; \
	echo "=== Running tests ==="; \
	$(MAKE) test; \
	echo ""; \
	echo "=== Dev setup complete ==="
