# AGENTS.md - ghdiff

CLI tool that displays git diffs in a GitHub-style web UI. Zero external Go
dependencies. Frontend is vanilla JS/CSS/HTML embedded into the binary via
`//go:embed`.

## Project layout

```
main.go              Entry point: CLI parsing, server startup, signal handling
internal/cli/        Command-line argument parsing (flag package), Config struct
internal/diff/       Unified diff parser (raw text -> structured types)
internal/git/        Git subprocess wrapper (diff, merge-base, commits)
internal/server/     HTTP server: API endpoints, token auth, static serving
internal/browser/    Cross-platform browser opener (xdg-open/open/cmd)
web/                 Frontend static assets (HTML, CSS, JS) + embed.go
web/vendor/          Vendored highlight.js + GitHub dark CSS
```

## Build, test, lint

```sh
make check           # lint + test (run this before submitting)
make test            # go test ./...
make lint            # golangci-lint run ./...
make fmt             # golangci-lint fmt ./... (goimports)
make build           # go build -o ghdiff .
```

### Running a single test

```sh
go test ./internal/diff/ -run TestParse
go test ./internal/server/ -run TestAPIDiff
go test -run TestIntegrationGitMode   # integration test (from root)
```

Integration tests in `integration_test.go` build the binary and start a
subprocess. They are skipped with `go test -short ./...`.

### Test patterns

- Standard `testing` package only - no third-party test libraries
- White-box tests (same package): `package diff`, `package server`, etc.
- Table-driven tests with descriptive names for parser tests
- Subtests via `t.Run()` in integration tests
- `t.Helper()` on all test helper functions
- `testing/fstest.MapFS` for mock filesystems in server tests
- `net/http/httptest` for HTTP handler tests

## Code style

### Go

Formatter: `goimports` (via `golangci-lint fmt`). Linter: `golangci-lint` v2.
Run `make check` to verify. The lint config is in `.golangci.yml`.

Enabled linters: `errcheck` (with `check-type-assertions: true`), `govet`,
`staticcheck`, `unused`, `ineffassign`, `gocritic` (diagnostic, style,
performance), `revive` (blank-imports, exported, unreachable-code,
unused-parameter).

### Imports

Two groups separated by a blank line: stdlib first, then internal packages.
There are zero third-party dependencies.

```go
import (
    "fmt"
    "net/http"

    "github.com/lundberg/ghdiff/internal/diff"
    "github.com/lundberg/ghdiff/internal/server"
)
```

### Naming

- **Files**: lowercase, short, single-word preferred (`parser.go`, `types.go`,
  `embed.go`). Multi-word uses underscores (`cli_test.go`).
- **Packages**: single concept, match directory (`cli`, `diff`, `git`, `server`,
  `browser`).
- **Exported**: PascalCase (`ParseArgs`, `NewRepo`, `FileDiff`, `GetDiff`).
- **Unexported**: camelCase (`stdinDiff`, `writeJSON`, `authHeaders`).
- **Receivers**: short, one letter (`r *Repo`, `s *Server`, `f *flags`).
- **Test functions**: `TestXxx` or `TestXxx_Variant` with underscore subtypes
  (`TestParseArgs_DefaultConfig`).
- **JSON tags**: camelCase with omitempty where appropriate
  (`json:"oldName"`, `json:"newNum,omitempty"`).
- **Frontend JS**: camelCase functions (`renderFileTree`, `fetchDiff`).
- **Frontend CSS**: kebab-case classes (`file-tree`, `diff-content`),
  CSS variables (`--bg`, `--font-mono`).

### Error handling

- Always wrap errors with context: `fmt.Errorf("doing thing: %w", err)`.
- Use the `run()` pattern: `main()` calls `run() error`, prints errors to
  stderr, exits with code 1.
- Sentinel errors for control flow: `var ErrHelp = errors.New("help requested")`
  checked with `errors.Is()`.
- HTTP errors via `http.Error()` with appropriate status codes.
- Intentionally ignored errors marked explicitly: `_ = httpServer.Close()`.
- `panic` only for truly unrecoverable failures (e.g. `crypto/rand`).
- Input validation before git operations: `validateRef()` rejects refs
  starting with `-`.

### Types

- Struct types with JSON tags for API serialization (`diff.Result`,
  `diff.FileDiff`, `git.Commit`, `cli.Config`).
- Pointer receivers for methods with state.
- Stdlib interfaces used where applicable (`fs.FS`, `io.Writer`,
  `http.Handler`).
- `any` for generic JSON serialization (`func writeJSON(w http.ResponseWriter, v any)`).

### Frontend

- Vanilla JavaScript: IIFE pattern, `"use strict"`, no framework, no build step.
- HTML/CSS/JS are embedded into the Go binary via `web/embed.go`.
- Vendored highlight.js for syntax highlighting.
- No frontend build pipeline or bundler.

## Security considerations

- CSRF tokens for API requests (secure random, constant-time comparison).
- Ref validation prevents command injection via git arguments.
- Warning printed when binding to non-localhost addresses.

## Commit style

Lowercase imperative prefix: `feat:`, `fix:`, `refactor:`, `security:`, or
plain imperative (e.g. `add file tree navigation`).

## Task tracking

This project uses Beads (`bd`) for task tracking. Run `bd onboard` to
understand current project state and available issues before starting work.
