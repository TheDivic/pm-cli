GO  ?= go
BIN := bin/pm
PKG := ./...

# Build metadata injected into the version command.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE  := github.com/TheDivic/plaintext-projects/internal/cli
LDFLAGS := -s -w \
	-X $(MODULE).version=$(VERSION) \
	-X $(MODULE).commit=$(COMMIT) \
	-X $(MODULE).date=$(DATE)

.PHONY: all check fmt fmt-check vet lint test test-race build tidy clean

# Default: run the full gate, then build.
all: check build

# The same gate CI runs. Keep local and CI in lockstep.
check: fmt-check vet lint test-race

# Rewrite files to canonical Go formatting.
fmt:
	gofmt -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || true

# Fail if any file is not gofmt-clean.
fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then \
		echo "gofmt needed for:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet $(PKG)

lint:
	golangci-lint run

test:
	$(GO) test $(PKG)

test-race:
	$(GO) test -race $(PKG)

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/pm

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin
