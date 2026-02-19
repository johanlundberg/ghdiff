# gitdiffview Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a single-binary Go CLI tool that serves a GitHub-style diff viewer on localhost.

**Architecture:** Go binary embeds HTML/CSS/JS frontend. CLI parses args, shells out to git for diff data, serves HTTP API + static assets. Frontend fetches diff JSON and renders GitHub-style file tree + diff view.

**Tech Stack:** Go stdlib (net/http, os/exec, embed, flag, encoding/json), Highlight.js (vendored), vanilla HTML/CSS/JS.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/cli/cli.go`

**Step 1: Initialize Go module**

```bash
go mod init github.com/lundberg/gitdiffview
```

**Step 2: Create main.go entry point**

Create `main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/lundberg/gitdiffview/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Create CLI argument parser**

Create `internal/cli/cli.go` with:
- Parse positional args: none (default merge-base), one (commit), two (ref1 ref2), `-` (stdin)
- Parse flags: `--port`, `--host`, `--no-open`, `--mode`
- Return a `Config` struct with all parsed values
- Detect stdin mode: explicit `-` arg, or piped stdin when no positional args/flags conflict
- Detect main branch: try `git rev-parse --verify main` then `master`, error if neither exists
- Compute merge-base: `git merge-base HEAD <main-branch>`

Config struct:

```go
type Config struct {
	Mode      string // "merge-base", "commit", "compare", "working", "stdin"
	Base      string // base ref for diff
	Target    string // target ref (or empty for working tree)
	Port      int
	Host      string
	NoOpen    bool
	ViewMode  string // "split" or "unified"
}
```

**Step 4: Verify it compiles**

```bash
go build -o gitdiffview .
```

Expected: binary compiles, running `./gitdiffview --help` shows usage.

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: project scaffolding with CLI arg parser"
```

---

### Task 2: Unified Diff Parser

**Files:**
- Create: `internal/diff/parser.go`
- Create: `internal/diff/parser_test.go`
- Create: `internal/diff/types.go`

**Step 1: Define diff data types**

Create `internal/diff/types.go`:

```go
package diff

type DiffResult struct {
	Files []FileDiff `json:"files"`
}

type FileDiff struct {
	OldName  string `json:"oldName"`
	NewName  string `json:"newName"`
	Status   string `json:"status"` // "added", "deleted", "modified", "renamed"
	IsBinary bool   `json:"isBinary"`
	Hunks    []Hunk `json:"hunks"`
}

type Hunk struct {
	OldStart int    `json:"oldStart"`
	OldLines int    `json:"oldLines"`
	NewStart int    `json:"newStart"`
	NewLines int    `json:"newLines"`
	Header   string `json:"header"`
	Lines    []Line `json:"lines"`
}

type Line struct {
	Type    string `json:"type"` // "add", "delete", "context"
	Content string `json:"content"`
	OldNum  int    `json:"oldNum,omitempty"`
	NewNum  int    `json:"newNum,omitempty"`
}
```

**Step 2: Write failing tests for the parser**

Create `internal/diff/parser_test.go` with test cases:
- Parse a simple file modification (one hunk, added + deleted lines)
- Parse a new file (all additions)
- Parse a deleted file (all deletions)
- Parse a renamed file
- Parse multiple files in one diff
- Parse binary file indication
- Parse empty diff (no files)
- Handle hunk headers with function context (e.g., `@@ -10,6 +10,8 @@ func main()`)

Use table-driven tests. Each test provides raw unified diff text as input and expected `DiffResult` as output.

Run: `go test ./internal/diff/ -v`
Expected: all tests FAIL (parser not implemented yet).

**Step 3: Implement the parser**

Create `internal/diff/parser.go`:
- `func Parse(input string) (*DiffResult, error)` - main entry point
- Parse `diff --git a/... b/...` headers to extract file names
- Parse `--- a/...` and `+++ b/...` to detect add/delete (when `/dev/null`)
- Parse `@@ -old,count +new,count @@` hunk headers
- Parse `+`, `-`, ` ` prefixed lines within hunks
- Track line numbers for old and new sides
- Detect `rename from`/`rename to` for renames
- Detect `Binary files` line for binary files
- Handle edge cases: no newline at end of file, empty hunks

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/diff/ -v`
Expected: all tests PASS.

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: unified diff parser with comprehensive tests"
```

---

### Task 3: Git Integration

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

**Step 1: Write failing tests**

Create `internal/git/git_test.go`. Tests will create temporary git repos using `os.MkdirTemp` and `os/exec` to run git commands, then verify our functions return correct results:
- `TestGetMainBranch` - detects `main` or `master`
- `TestGetMergeBase` - computes merge-base between HEAD and main
- `TestGetDiff` - runs git diff and returns unified diff text
- `TestGetCommits` - returns recent commits with hash, message, author, date

Run: `go test ./internal/git/ -v`
Expected: FAIL.

**Step 2: Implement git operations**

Create `internal/git/git.go`:

```go
package git

type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// GetMainBranch returns "main" or "master", whichever exists
func GetMainBranch() (string, error)

// GetMergeBase returns the merge-base commit between two refs
func GetMergeBase(ref1, ref2 string) (string, error)

// GetDiff returns unified diff text between two refs (or vs working tree)
func GetDiff(base string, target string) (string, error)

// GetCommits returns recent commits (limit n) for the current branch
func GetCommits(n int) ([]Commit, error)
```

All functions use `os/exec.Command("git", ...)` to shell out.
`GetDiff` with empty target means diff against working tree (includes staged + unstaged).

**Step 3: Run tests**

Run: `go test ./internal/git/ -v`
Expected: PASS.

**Step 4: Commit**

```bash
git add -A && git commit -m "feat: git integration for diff, merge-base, and commits"
```

---

### Task 4: HTTP Server + API

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`

**Step 1: Write failing tests**

Create `internal/server/server_test.go`:
- `TestAPICommits` - `GET /api/commits` returns JSON array of commits
- `TestAPIDiff` - `GET /api/diff` returns JSON diff result
- `TestAPIDiffWithBase` - `GET /api/diff?base=abc123` returns diff against specific commit
- `TestStaticServing` - `GET /` returns HTML content

Use `httptest.NewServer` for tests.

Run: `go test ./internal/server/ -v`
Expected: FAIL.

**Step 2: Implement the server**

Create `internal/server/server.go`:

```go
package server

type Server struct {
	config  *cli.Config
	mux     *http.ServeMux
}

func New(config *cli.Config) *Server

// Start starts the HTTP server and optionally opens browser
func (s *Server) Start() error

// Handlers:
// GET /           - serve embedded frontend (index.html)
// GET /api/diff   - return parsed diff JSON (optional ?base= query param)
// GET /api/commits - return recent commits JSON
```

- The diff handler: if `?base=` param provided, re-run `git diff` with that base; otherwise use config default
- For stdin mode: diff data is read once at startup, `?base=` is ignored
- The commits handler: call `git.GetCommits(50)`
- Static handler: serve embedded files (placeholder for now, will be wired in Task 6)

**Step 3: Run tests**

Run: `go test ./internal/server/ -v`
Expected: PASS.

**Step 4: Commit**

```bash
git add -A && git commit -m "feat: HTTP server with diff and commits API endpoints"
```

---

### Task 5: Frontend - HTML/CSS/JS

**Files:**
- Create: `web/index.html`
- Create: `web/css/style.css`
- Create: `web/js/app.js`

This is the largest task. The frontend is vanilla HTML/CSS/JS (no build step, no framework).

**Step 1: Create the HTML skeleton**

Create `web/index.html`:
- Top bar with commit picker dropdown and split/unified toggle
- Left panel (file tree)
- Right panel (diff content area)
- Script and CSS includes
- Include Highlight.js from CDN for now (will vendor later)

**Step 2: Create the CSS**

Create `web/css/style.css` - GitHub dark theme:
- Background: `#0d1117` (GitHub dark bg)
- Surface: `#161b22` (panels, cards)
- Border: `#30363d`
- Text: `#e6edf3`
- Additions: `#1a4721` bg, `#3fb950` text/border
- Deletions: `#67060c` bg, `#f85149` text/border
- File tree: fixed left panel, ~280px wide
- Diff content: scrollable right panel
- Split view: two-column table layout per hunk
- Unified view: single-column table layout
- Line numbers: `#484f58` color, monospace
- Hunk headers: `#1f2937` bg
- Top bar: sticky, `#161b22` bg
- File headers: sticky below top bar, `#161b22` bg, filename + status badge
- Scrollbar styling matching GitHub dark

**Step 3: Create the JavaScript**

Create `web/js/app.js`:

Functions needed:
- `fetchDiff(base?)` - GET `/api/diff` or `/api/diff?base=X`, store result
- `fetchCommits()` - GET `/api/commits`, populate dropdown
- `renderFileTree(files)` - build nested folder tree in left panel
  - Group files by directory path
  - Create expandable folder nodes
  - File nodes show status icon (green + for added, yellow ~ modified, red - deleted)
  - Click handler scrolls to file in right panel
- `renderDiffContent(files)` - render all file diffs stacked in right panel
  - Each file: collapsible section with header (filename + status + stats)
  - Render hunks with line numbers
  - Split view: two-column layout, align old/new lines side by side
  - Unified view: single column, +/- prefixed lines
  - Apply syntax highlighting via Highlight.js after render
- `toggleViewMode(mode)` - switch between split/unified, re-render
- `selectCommit(hash)` - re-fetch diff with new base, re-render everything
- `scrollToFile(filename)` - smooth scroll to file section, highlight in tree
- `buildFileTree(files)` - convert flat file paths to nested tree structure

Split view alignment logic:
- Walk through hunk lines
- Context lines: show on both sides
- Add lines: show on new side, blank on old side  
- Delete lines: show on old side, blank on new side
- Adjacent delete+add pairs: show side by side (as modifications)

**Step 4: Verify manually**

Serve `web/` directory with a simple file server and test with sample data. This will be properly integrated in Task 6.

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: GitHub-style dark theme frontend with file tree and diff views"
```

---

### Task 6: Embed Frontend + Wire Everything Together

**Files:**
- Create: `web/embed.go`
- Modify: `internal/server/server.go` - wire embedded filesystem
- Modify: `main.go` - wire CLI → server startup
- Modify: `internal/cli/cli.go` - add browser open logic

**Step 1: Create embed.go**

Create `web/embed.go`:

```go
package web

import "embed"

//go:embed index.html css/* js/*
var Assets embed.FS
```

**Step 2: Wire embedded FS into server**

Modify `internal/server/server.go`:
- Import `web` package
- Serve `web.Assets` via `http.FileServer(http.FS(web.Assets))` at `/`

**Step 3: Wire CLI → server in main.go**

- `cli.Run()` parses config
- If stdin mode, read all stdin into a string
- Create server with config + optional stdin diff data
- Server starts, prints URL, opens browser (unless --no-open)
- Block until Ctrl+C (signal handling with `os/signal`)

**Step 4: Add browser open logic**

In `internal/cli/cli.go` or a new `internal/browser/open.go`:
- `func OpenBrowser(url string) error`
- Use `xdg-open` on Linux, `open` on macOS, `start` on Windows
- Detect OS via `runtime.GOOS`

**Step 5: Build and test end-to-end**

```bash
go build -o gitdiffview .
```

Test in an actual git repo:
- `./gitdiffview` - should open browser with diff view
- `./gitdiffview --no-open --port 8080` - should print URL without opening
- `echo "some diff" | ./gitdiffview -` - stdin mode

**Step 6: Commit**

```bash
git add -A && git commit -m "feat: embed frontend assets and wire CLI → server → browser"
```

---

### Task 7: Vendor Highlight.js + Polish

**Files:**
- Create: `web/vendor/highlight.min.js`
- Create: `web/vendor/github-dark.min.css`
- Modify: `web/index.html` - update script/CSS refs to vendored files
- Modify: `web/embed.go` - include vendor dir

**Step 1: Download Highlight.js custom build**

Download a custom Highlight.js build with common languages:
- Go, JavaScript, TypeScript, Python, Rust, Java, C, C++, Ruby, Shell, SQL, JSON, YAML, XML, Markdown, CSS, HTML, Dockerfile, Makefile, TOML

Download the minified JS + GitHub Dark CSS theme.

**Step 2: Update embed.go**

```go
//go:embed index.html css/* js/* vendor/*
var Assets embed.FS
```

**Step 3: Update index.html references**

Change CDN links to `/vendor/highlight.min.js` and `/vendor/github-dark.min.css`.

**Step 4: Build and verify**

```bash
go build -o gitdiffview .
```

Verify: binary works offline (no CDN requests), syntax highlighting works.

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: vendor highlight.js for offline syntax highlighting"
```

---

### Task 8: End-to-End Testing + Edge Cases

**Files:**
- Create: `internal/integration_test.go`
- Modify: `internal/diff/parser.go` - fix any edge cases found
- Modify: `web/js/app.js` - fix any rendering issues

**Step 1: Write integration tests**

Create integration tests that:
- Create a temp git repo with known commits
- Run gitdiffview as a subprocess with `--no-open`
- Hit the API endpoints and verify JSON responses
- Verify static assets are served
- Test stdin mode
- Test various CLI arg combinations

**Step 2: Test edge cases in diff parser**

Add parser tests for:
- Files with spaces in names
- Very long lines
- Empty files
- Files with only whitespace changes
- No newline at end of file (`\ No newline at end of file`)
- Diffs with 0-context lines

**Step 3: Fix any issues found**

**Step 4: Final build + verify binary size**

```bash
go build -ldflags="-s -w" -o gitdiffview .
ls -lh gitdiffview
```

Expected: binary ~10-15 MB.

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: integration tests and edge case handling"
```
