GO  ?= go
BIN := bin/pm
PKG := ./...

# Build metadata injected into the version command.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE  := github.com/TheDivic/pm-cli/internal/cli
LDFLAGS := -s -w \
	-X $(MODULE).version=$(VERSION) \
	-X $(MODULE).commit=$(COMMIT) \
	-X $(MODULE).date=$(DATE)

.PHONY: all check fmt fmt-check vet lint test test-race build install demo tidy clean

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

# Install pm from local source to $(go env GOPATH)/bin. Carries no ldflags,
# same as `go install .../cmd/pm@latest`; buildMetadata falls back to Go's own
# module and VCS build info in that case (see internal/cli/version.go).
install:
	$(GO) install ./cmd/pm

# Re-record the README demo. Needs charmbracelet/vhs on PATH; writes
# docs/demo.gif, which is committed so GitHub can serve it.
demo: build
	@command -v vhs >/dev/null 2>&1 || { \
		echo "vhs is required to record the demo: https://github.com/charmbracelet/vhs"; \
		exit 1; }
	PATH="$(CURDIR)/bin:$$PATH" vhs docs/demo.tape

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin
