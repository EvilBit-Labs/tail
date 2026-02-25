# Proposal: EvilBit-Labs Tail Library

## Summary

Modernize the existing `github.com/EvilBit-Labs/tail` fork — an actively
maintained, cross-platform Go library for tailing files. Replace the abandoned
`gopkg.in/tomb.v1` concurrency primitive with stdlib `context.Context`, maintain
the existing public API for drop-in compatibility, and provide ongoing security
and maintenance support.

## Motivation

### The ecosystem gap

There is no actively maintained, cross-platform, context-native tail library in
Go today:

| Library | Status | Last Release | context.Context | Windows | Stars |
|---------|--------|-------------|-----------------|---------|-------|
| nxadm/tail | Inactive (12+ mo) | v1.4.11 (2023) | No (tomb.v1) | Yes | ~500 |
| go-faster/tail | Archived (Nov 2023) | v0.3.0 (Jan 2022) | Yes | No | 23 |
| jdrews/go-tailer | Active | v1.2.1 (Sep 2025) | No | Yes | 2 |
| hpcloud/tail | Abandoned | 2015 | No (tomb.v1) | Yes | ~2.6k |

`nxadm/tail` itself was a fork of the abandoned `hpcloud/tail`. Both depend on
`gopkg.in/tomb.v1`, an abandoned goroutine lifecycle manager that predates
`context.Context` (added in Go 1.7, 2016). No future security patches are
expected for either `nxadm/tail` or `tomb.v1`.

`go-faster/tail` solved the tomb→context problem but was archived and never
supported Windows.

### Why this fork

Many Go projects depend on `nxadm/tail` for reliable file tailing. With the
upstream inactive and no maintained alternative offering cross-platform support
with modern Go idioms, the community is left choosing between an abandoned
dependency and writing their own.

This fork aims to fill that gap: a drop-in replacement that modernizes the
internals while preserving the stable API that existing users rely on.

## Technical Plan

### Approach: Fork, not rewrite

This repo is already a fork of `nxadm/tail`, preserving:

- Git history and attribution chain (hpcloud → nxadm → EvilBit-Labs)
- Existing test suite and platform-specific code (inotify, kqueue, polling,
  Windows syscalls)
- Public API compatibility — users change one import path
- GitHub fork network discoverability

### Phase 1: Foundation (Week 1)

**Goal**: Publishable fork with zero functional changes.

- [ ] Update `go.mod` module path to `github.com/EvilBit-Labs/tail`
- [ ] Update internal import paths (`ratelimiter`, `util`, `watch`, `winfile`)
- [ ] Upgrade Go version in `go.mod` to 1.22+
- [ ] Run `go mod tidy` to clean up dependencies
- [ ] Set up CI (GitHub Actions: lint, test, race detection on Linux/macOS/Windows)
- [ ] Update README: maintenance status, migration guide, attribution
- [ ] Update LICENSE: preserve existing MIT license, add EvilBit-Labs copyright
- [ ] Add CONTRIBUTING.md and SECURITY.md
- [ ] Tag `v1.5.0` (signals continuation from nxadm's v1.4.11)
- [ ] Verify `go get github.com/EvilBit-Labs/tail@v1.5.0` works

### Phase 2: tomb.v1 → context.Context (Weeks 2-3)

**Goal**: Remove `gopkg.in/tomb.v1` entirely. This is the core modernization.

`tomb.Tomb` provides three things that need stdlib replacements:

| tomb.v1 | stdlib replacement |
|---------|-------------------|
| `tomb.Dying` channel | `ctx.Done()` channel |
| `tomb.Kill(err)` | `cancel()` (or `context.CancelCauseFunc`) |
| `tomb.Err()` | `context.Cause(ctx)` (Go 1.20+) |
| `tomb.ErrDying` sentinel | `context.Canceled` |

Changes:

- [ ] `Tail` struct: replace `tomb.Tomb` field with `ctx context.Context` +
  `cancel context.CancelCauseFunc`
- [ ] `TailFile()` → accept optional `context.Context` (backward-compatible:
  `nil` ctx defaults to `context.Background()`)
- [ ] `Tail.Stop()` → calls `cancel(ErrStopped)` instead of `tomb.Kill(nil)`
- [ ] `Tail.Done()` → returns `ctx.Done()` (same channel semantics)
- [ ] `Tail.Err()` → returns `context.Cause(ctx)`
- [ ] Internal goroutines: replace `tomb.Dying` select cases with `ctx.Done()`
- [ ] `watch/` package: thread context through inotify/kqueue/polling watchers
- [ ] Remove `gopkg.in/tomb.v1` from `go.mod`
- [ ] Remove `tomb.ErrDying` references (replace with `context.Canceled` checks)
- [ ] Full test pass with `-race` on all three platforms
- [ ] Tag `v2.0.0` (breaking change: tomb removal from exported Tail struct)

**Reference**: `go-faster/tail` already solved tomb→context for Linux/macOS.
Their approach can inform the migration without direct code copying.

**Semver note**: `tomb.Tomb` is exported on the `Tail` struct, so removing it is
a breaking change → major version bump to v2. The alternative is keeping v1.x
with a `context.Context`-accepting `TailFileContext()` function alongside the
existing API, deferring the v2 break.

### Phase 3: Modernization (Week 4)

**Goal**: Idiomatic modern Go.

- [ ] Minimum Go version: 1.22 (for `context.AfterFunc`, improved generics)
- [ ] Replace `log.Logger` interface with `slog.Logger` (Go 1.21+)
- [ ] Add `TailFileContext(ctx, path, config)` convenience function
- [ ] Audit and simplify `ratelimiter/` package (or remove if unused by most
  consumers)
- [ ] Audit `winfile/` package for modern Windows API usage
- [ ] Add benchmarks (compare against `go-faster/tail` numbers)
- [ ] Add `MIGRATION.md` for nxadm/tail users

### Phase 4: Ongoing Maintenance

- Dependabot / Renovate for dependency updates
- Security advisory monitoring
- Respond to issues/PRs within 1 week
- Quarterly review of Go version support (drop old versions per Go release
  policy)
- Estimated effort: ~2-4 hours/month steady-state

## API Compatibility

### Drop-in migration for v1.x

```go
// Before
import "github.com/nxadm/tail"

// After
import "github.com/EvilBit-Labs/tail"
```

No code changes required. `tail.TailFile`, `tail.Config`, `tail.Line` — all
identical.

### v2.x context-native API

```go
// New context-aware function
tailer, err := tail.TailFileContext(ctx, "/var/log/app.log", tail.Config{
    Follow: true,
})
defer tailer.Stop()

for line := range tailer.Lines {
    fmt.Println(line.Text)
}
```

The `Lines` channel closes when the context is cancelled, enabling clean
goroutine shutdown without tomb's `Dying`/`Dead` ceremony.

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Maintenance burnout | Medium | Scope strictly to tail functionality; reject feature creep |
| Windows-specific bugs | Low | CI on windows-latest; community bug reports |
| Breaking existing users on v2 | Low | v1.x branch stays available; v2 migration guide |
| tomb→context migration introduces bugs | Medium | Existing test suite; race detector; go-faster/tail as reference |

## Level of Effort

| Phase | Effort | Timeline |
|-------|--------|----------|
| 1. Foundation | 4-6 hours | Week 1 |
| 2. tomb→context | 16-24 hours | Weeks 2-3 |
| 3. Modernization | 8-12 hours | Week 4 |
| 4. Ongoing | 2-4 hours/month | Indefinite |
| **Total initial** | **~30-40 hours** | **~4 weeks** |

## Decision: Fork vs. Fresh Repo

**Decision: Fork.** (Already done — this repo is the fork.)

| Factor | Fork | Fresh |
|--------|------|-------|
| Drop-in import path swap | Yes | No (new API) |
| Git history / attribution | Preserved | Must credit manually |
| Existing tests + platform code | Inherited | Must rewrite |
| GitHub fork network discovery | Yes | No |
| Legacy baggage | tomb.v1 in history | Clean |
| API freedom | Constrained by compat | Full freedom |

Forking preserves the attribution chain, inherits the existing test suite, and
makes the project discoverable through GitHub's fork network to anyone already
using `nxadm/tail`.

## Acknowledgments

This project builds on the excellent work of the original authors:

- [hpcloud/tail](https://github.com/hpcloud/tail) — the original Go tail
  library
- [nxadm/tail](https://github.com/nxadm/tail) — revamped fork that kept the
  library alive for years
- [go-faster/tail](https://github.com/go-faster/tail) — pioneered the
  tomb→context migration (Linux/macOS)

## References

- [gopkg.in/tomb.v1](https://github.com/go-tomb/tomb) — concurrency primitive
  being replaced
- [context.CancelCauseFunc](https://pkg.go.dev/context#CancelCauseFunc) —
  Go 1.20+ stdlib replacement for tomb error propagation
