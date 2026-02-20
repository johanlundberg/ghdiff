# ghdiff Design

## Overview

A single-binary CLI tool written in Go that displays git diffs in a GitHub-style web UI.
It shells out to `git` to generate diffs (or accepts unified diff via stdin), parses the
output, and serves an embedded web page on localhost.

## Default Behavior

- No args: compute `git merge-base HEAD main` (or `master`) and diff working tree against it
- Shows all changes on the current branch compared to where it diverged from main
- Mirrors what you'd see in a GitHub PR "Files changed" view

## CLI Interface

```
ghdiff                        # diff working tree vs merge-base with main
ghdiff <commit>               # diff working tree vs specific commit
ghdiff <ref1> <ref2>          # compare two refs
ghdiff .                      # uncommitted changes only
cat patch.diff | ghdiff -     # stdin mode
```

Flags: `--port` (default 4966), `--host` (default 127.0.0.1), `--no-open`, `--mode` (split/unified)

## Architecture

```
CLI (args/stdin) → Diff Source (git/stdin) → Unified Diff Parser → HTTP Server → Embedded SPA
```

### Components

1. **CLI layer** - `flag` package, argument parsing
2. **Diff source** - shells out to `git diff` or reads stdin
3. **Unified diff parser** - pure Go, parses into structured data (files → hunks → lines)
4. **HTTP server** - `net/http` stdlib
   - `GET /` - serves embedded SPA
   - `GET /api/diff?base=<commit>` - returns parsed diff as JSON
   - `GET /api/commits` - returns recent commits for the picker
5. **Frontend** - embedded HTML/CSS/JS via `//go:embed`

## UI Layout

GitHub PR "Files changed" style:

- **Top bar**: commit picker dropdown + split/unified toggle
- **Left panel**: nested file tree with folders that expand/collapse, files show status (added/modified/deleted)
- **Right panel**: all file diffs stacked vertically, clicking a file in the tree scrolls to it
- **Theme**: GitHub dark theme colors
- **Views**: split (side-by-side) and unified with toggle
- **Syntax highlighting**: Highlight.js (vendored/embedded)

## Commit Picker

- Shows current base: "Comparing against: abc1234 (merge-base with main)"
- Dropdown lists recent commits (`git log --oneline -n 50`)
- Selecting a commit re-diffs via `GET /api/diff?base=<commit>`

## Dependencies

- External: only `git` in PATH
- Go modules: near-zero third-party deps (stdlib covers everything)
- Frontend: Highlight.js vendored for syntax highlighting, vanilla HTML/CSS/JS

## Out of Scope

- Comment/review system
- GitHub PR integration
- TUI mode
- WebSocket live reload
