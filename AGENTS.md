# AGENTS.md — tail

## Project Description

Go library for tailing files with support for log rotation, originally forked
from [nxadm/tail](https://github.com/nxadm/tail). Provides an API for reading
lines from a file as they are appended, similar to `tail -f`.

## Module Path

```text
github.com/nxadm/tail
```

> **Note:** The module path will change to `github.com/EvilBit-Labs/tail` in
> Phase 1. Do NOT modify import paths yet.

## Package Structure

| Package        | Description                                                                               |
| -------------- | ----------------------------------------------------------------------------------------- |
| `tail` (root)  | Core library — `Tail` struct, `Config`, line reading                                      |
| `ratelimiter/` | Leaky bucket rate limiter for throttling tail output                                      |
| `util/`        | Internal utilities (temp file helpers, logging)                                           |
| `watch/`       | File watcher abstraction (`FileWatcher` interface) and implementations (polling, inotify) |
| `winfile/`     | Windows-specific file open wrapper                                                        |
| `cmd/gotail/`  | CLI tool — standalone `tail -f` replacement                                               |

## Build and Test

```sh
# Build
go build ./...

# Test (with race detector)
go test -race -v -timeout 2m ./...

# Lint
golangci-lint run
```

## Key Dependencies

| Dependency                     | Purpose                                                                |
| ------------------------------ | ---------------------------------------------------------------------- |
| `github.com/fsnotify/fsnotify` | Cross-platform file system notifications                               |
| `gopkg.in/tomb.v1`             | Goroutine lifecycle management (to be replaced with `context.Context`) |

## Platform Support

Linux, macOS, Windows, FreeBSD.

## Coding Conventions

- `tomb.Tomb` is embedded in the `Tail` struct — controls goroutine lifecycle
  and shutdown signaling.
- `watch.FileWatcher` is the watcher abstraction — implementations live in
  `watch/` (inotify watcher, polling watcher).
- Follow existing patterns in the codebase before introducing new abstractions.
- Keep the public API surface minimal and backward-compatible.

## CI

- **GitHub Actions**: Multi-OS (Linux, macOS, Windows) and multi-Go-version
  matrix.
- **Cirrus CI**: FreeBSD builds.

## Gotchas

- `go.mod` specifies `go 1.18` — some modern linters may still be disabled due to version constraint
- golangci-lint reports many existing issues in upstream code — config is correct, code needs incremental cleanup
- Tests do real file I/O with timing-dependent operations (~10s runtime) — always use `-timeout 2m`
- `vendor/` directory is checked in — run `go mod vendor` after dependency changes
- Default branch is `master`, not `main`
