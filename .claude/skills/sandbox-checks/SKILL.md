---
name: sandbox-checks
description: Run this repository's verification gate (make check — gofmt, go vet, golangci-lint, go test -race) and build the pm binary correctly inside a Docker sandbox. Use before committing, when make check fails on missing golangci-lint or a cgo/C-compiler error, when a build needs smoke-testing, or when a piped make check reports a failure the log does not explain.
---

# Running checks and builds in this repo

`make check` is the gate CI runs — `gofmt`, `go vet`, `golangci-lint`, and
`go test -race`, in that order. Keep local and CI in lockstep; never commit
without it passing.

In a **fresh Docker sandbox `make check` fails for two reasons that have nothing
to do with your change.** Fix both once per sandbox, then forget about it.

## One-time sandbox setup

```sh
export PATH="$(go env GOPATH)/bin:$PATH"     # golangci-lint and goreleaser live here
sudo apt-get install -y gcc libc6-dev        # go test -race needs cgo
```

- **`make: golangci-lint: No such file or directory`** — the binary is installed
  at `$(go env GOPATH)/bin` but that is not on `PATH`. Export it as above; add the
  line to `/etc/sandbox-persistent.sh` to keep it across bash calls.
- **`-race requires cgo`, or `cgo: C compiler ... not found`** — install `gcc`.
- **`stdint.h: No such file or directory` / `errno.h: No such file or directory`**
  — `gcc` alone is not enough; you also need `libc6-dev` for the C headers.
  Install **both** in one go and skip the two-step rediscovery.

Then:

```sh
make check
```

## Do not build into ./bin

**Use `/tmp`, never `make build` or `make all`, while in a sandbox.**

```sh
go build -o /tmp/pm ./cmd/pm      # smoke tests
go run ./cmd/pm --root /tmp/x projects list
```

The working tree is the host's, mounted directly (confirm with
`[ -d /run/sandbox/source ] && echo clone || echo direct`). `make build` writes
`bin/pm` compiled for the sandbox's OS/arch, overwriting the host's binary with
something it cannot exec — the user then gets "exec format error" on their own
machine while you work. Leave `bin/` to the host's own `make build`.

`make all` runs `check` **and** `build`, so it has the same problem. Run
`make check` instead.

## A piped `make check` can report a phantom failure

```sh
make check | tail -20        # may print exit 1 even though every stage passed
```

That is SIGPIPE from `head`/`tail` closing the pipe, not a real failure. Before
believing it:

```sh
make check > /tmp/check.log 2>&1; echo "exit=$?"
```

Trust that exit code, and read the log for the actual stage output.

## Individual targets

`make fmt` (rewrite), `fmt-check`, `vet`, `lint`, `test`, `test-race`, `tidy`.
Use `make fmt` to fix formatting rather than hand-aligning code.

CI additionally runs `go build ./...` on Go 1.26 — if `make check` passes but CI
fails, compare against `.github/workflows/ci.yml` before assuming flakiness.
