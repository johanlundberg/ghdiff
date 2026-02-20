# ghdiff

A CLI tool that displays git diffs in a GitHub-style web UI. Runs a local
server and opens your browser to a dark-themed, interactive diff viewer -- like
the "Files changed" tab on a GitHub pull request, but for your local repo.

Zero external Go dependencies. Single binary. No build step for the frontend.

## Features

- **Split and unified diff views** with syntax highlighting
- **File tree sidebar** with collapsible folders and color-coded status indicators
- **Commit picker** dropdowns to dynamically switch base and target refs
- **Stdin support** for piping any unified diff
- **Auto-opens browser** on startup (disable with `--no-open`)
- **Secure by default** -- CSRF-protected API, localhost-only binding

## Install

```sh
go install github.com/lundberg/ghdiff@latest
```

Or build from source:

```sh
git clone https://github.com/lundberg/ghdiff.git
cd ghdiff
make build    # produces ./ghdiff
```

Requires `git` in your PATH (except when reading from stdin).

## Usage

```
ghdiff [flags] [ref1 [ref2]]
```

### Examples

```sh
# Review current branch vs main (like a GitHub PR)
ghdiff

# Uncommitted changes (staged + unstaged)
ghdiff .

# View a single commit
ghdiff HEAD~1

# Compare two refs
ghdiff main feature-branch
ghdiff v1.0.0 v2.0.0

# Pipe any unified diff
git diff HEAD~3 | ghdiff -
cat changes.patch | ghdiff -
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `0` (auto) | HTTP server port |
| `--host` | `localhost` | HTTP server host |
| `--no-open` | `false` | Don't open browser automatically |
| `--mode` | `split` | Initial view mode: `split` or `unified` |

### Modes

| Arguments | Mode | Description |
|-----------|------|-------------|
| _(none)_ | merge-base | Diff working tree against merge-base with main/master |
| `.` | working | Uncommitted changes vs HEAD |
| `<commit>` | commit | Diff working tree against a specific commit |
| `<ref1> <ref2>` | compare | Diff between two refs |
| `-` | stdin | Read unified diff from stdin |

## How it works

`ghdiff` starts a local HTTP server that serves an embedded single-page
application. The frontend fetches diff data from a JSON API and renders it in
the browser. In git mode, the server shells out to `git diff` and parses the
output on each request, so changing refs via the commit picker dropdowns shows
live results. In stdin mode, the diff is parsed once at startup.

The HTML, CSS, and JavaScript are embedded into the Go binary at compile time
via `//go:embed`, so the final artifact is a single self-contained executable.

## Development

```sh
make check    # lint + test (run before submitting)
make test     # go test ./...
make lint     # golangci-lint run ./...
make fmt      # goimports via golangci-lint
make build    # go build -o ghdiff .
```

Run a single test:

```sh
go test ./internal/diff/ -run TestParse
go test ./internal/server/ -run TestAPIDiff
go test -run TestIntegrationGitMode
```

Integration tests build the binary and start it as a subprocess. They are
skipped with `go test -short ./...`.

### Project structure

```
main.go              Entry point, server startup, signal handling
internal/cli/        CLI argument parsing, Config struct
internal/diff/       Unified diff parser
internal/git/        Git subprocess wrapper
internal/server/     HTTP server, API endpoints, auth
internal/browser/    Cross-platform browser opener
web/                 Embedded frontend (HTML, CSS, JS)
web/vendor/          Vendored highlight.js
```
