GO  ?= go
BIN := bin/pm
PKG := ./...

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
	$(GO) build -o $(BIN) ./cmd/pm

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin
