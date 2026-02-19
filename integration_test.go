package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lundberg/gitdiffview/internal/diff"
	"github.com/lundberg/gitdiffview/internal/git"
)

// initTestRepo creates a temporary git repo with user config and an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "branch", "-M", "main"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}
	return dir
}

// commitFile creates/overwrites a file and commits it. Returns the commit hash.
func commitFile(t *testing.T, dir, name, content, message string) string {
	t.Helper()

	// Ensure parent directory exists
	parent := filepath.Dir(filepath.Join(dir, name))
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}
	for _, args := range [][]string{
		{"git", "add", name},
		{"git", "commit", "-m", message},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("commit %v failed: %v\n%s", args, err, out)
		}
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// buildBinary builds the gitdiffview binary and returns the path.
// The binary is built once per test run via t.TempDir().
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "gitdiffview")

	// Get the module root (where go.mod is)
	// We need to build from the source directory
	sourceDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = sourceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binPath
}

var listenRe = regexp.MustCompile(`Listening on http://([^\s]+)`)

// startBinary starts the gitdiffview binary and waits for it to be ready.
// Returns the base URL and a cancel function. The process is killed when cancel is called.
func startBinary(t *testing.T, binPath string, dir string, args ...string) (string, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	fullArgs := append([]string{"--no-open", "--port", "0"}, args...)
	cmd := exec.CommandContext(ctx, binPath, fullArgs...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start binary: %v", err)
	}

	// Wait for "Listening on" message to get the URL
	scanner := bufio.NewScanner(stdout)
	urlCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if m := listenRe.FindStringSubmatch(line); m != nil {
				urlCh <- "http://" + m[1]
				return
			}
		}
	}()

	// Cleanup function that kills the process
	cleanup := func() {
		cancel()
		cmd.Wait()
	}

	select {
	case url := <-urlCh:
		return url, cleanup
	case <-time.After(10 * time.Second):
		cleanup()
		t.Fatal("timeout waiting for binary to start")
		return "", nil // unreachable
	}
}

// startBinaryStdin starts the binary in stdin mode, piping diffData to its stdin.
func startBinaryStdin(t *testing.T, binPath string, diffData string, extraArgs ...string) (string, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	args := append([]string{"--no-open", "--port", "0", "-"}, extraArgs...)
	cmd := exec.CommandContext(ctx, binPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start binary: %v", err)
	}

	// Write diff data to stdin and close
	go func() {
		io.WriteString(stdin, diffData)
		stdin.Close()
	}()

	scanner := bufio.NewScanner(stdout)
	urlCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if m := listenRe.FindStringSubmatch(line); m != nil {
				urlCh <- "http://" + m[1]
				return
			}
		}
	}()

	cleanup := func() {
		cancel()
		cmd.Wait()
	}

	select {
	case url := <-urlCh:
		return url, cleanup
	case <-time.After(10 * time.Second):
		cleanup()
		t.Fatal("timeout waiting for binary to start (stdin mode)")
		return "", nil
	}
}

func TestIntegrationGitMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)
	dir := initTestRepo(t)

	commitFile(t, dir, "hello.txt", "hello world\n", "initial commit")
	commitFile(t, dir, "hello.txt", "hello world\ngoodbye world\n", "add goodbye")

	baseURL, cleanup := startBinary(t, binPath, dir, "HEAD~1", "HEAD")
	defer cleanup()

	t.Run("api/diff returns valid JSON", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/diff")
		if err != nil {
			t.Fatalf("GET /api/diff: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}

		var result diff.DiffResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(result.Files) == 0 {
			t.Fatal("expected at least one file in diff")
		}
		if result.Files[0].NewName != "hello.txt" {
			t.Errorf("expected file name 'hello.txt', got %q", result.Files[0].NewName)
		}

		// Verify the diff content
		foundAdd := false
		for _, f := range result.Files {
			for _, h := range f.Hunks {
				for _, l := range h.Lines {
					if l.Type == "add" && l.Content == "goodbye world" {
						foundAdd = true
					}
				}
			}
		}
		if !foundAdd {
			t.Error("expected diff to contain added line 'goodbye world'")
		}
	})

	t.Run("api/commits returns commits", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/commits")
		if err != nil {
			t.Fatalf("GET /api/commits: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}

		var commits []git.Commit
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(commits) != 2 {
			t.Fatalf("expected 2 commits, got %d", len(commits))
		}
		if commits[0].Message != "add goodbye" {
			t.Errorf("expected first commit 'add goodbye', got %q", commits[0].Message)
		}
	})

	t.Run("static assets served at root", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		// Should serve HTML content
		if !strings.Contains(string(body), "<html") && !strings.Contains(string(body), "<!DOCTYPE") {
			t.Errorf("expected HTML content at root, got:\n%.200s", body)
		}
	})
}

func TestIntegrationStdinMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)

	diffData := `diff --git a/test.go b/test.go
index 1234567..abcdef0 100644
--- a/test.go
+++ b/test.go
@@ -1,3 +1,4 @@
 package main
 
 func main() {
+	fmt.Println("hello")
 }
`

	baseURL, cleanup := startBinaryStdin(t, binPath, diffData)
	defer cleanup()

	t.Run("api/diff returns stdin diff", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/diff")
		if err != nil {
			t.Fatalf("GET /api/diff: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var result diff.DiffResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(result.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Files))
		}
		if result.Files[0].NewName != "test.go" {
			t.Errorf("expected file name 'test.go', got %q", result.Files[0].NewName)
		}
	})

	t.Run("api/commits returns empty in stdin mode", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/commits")
		if err != nil {
			t.Fatalf("GET /api/commits: %v", err)
		}
		defer resp.Body.Close()

		var commits []git.Commit
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(commits) != 0 {
			t.Errorf("expected empty commits array in stdin mode, got %d", len(commits))
		}
	})
}

func TestIntegrationDiffWithBaseQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)
	dir := initTestRepo(t)

	firstHash := commitFile(t, dir, "file.txt", "line1\n", "first")
	commitFile(t, dir, "file.txt", "line1\nline2\n", "second")
	commitFile(t, dir, "file.txt", "line1\nline2\nline3\n", "third")

	// Start with HEAD~1..HEAD (only shows line3 change)
	baseURL, cleanup := startBinary(t, binPath, dir, "HEAD~1", "HEAD")
	defer cleanup()

	// Override base via query param to see all changes from first commit
	resp, err := http.Get(baseURL + "/api/diff?base=" + firstHash)
	if err != nil {
		t.Fatalf("GET /api/diff?base=...: %v", err)
	}
	defer resp.Body.Close()

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Should see both line2 and line3 as additions
	foundLine2 := false
	foundLine3 := false
	for _, f := range result.Files {
		for _, h := range f.Hunks {
			for _, l := range h.Lines {
				if l.Type == "add" && l.Content == "line2" {
					foundLine2 = true
				}
				if l.Type == "add" && l.Content == "line3" {
					foundLine3 = true
				}
			}
		}
	}
	if !foundLine2 {
		t.Error("expected diff from first commit to contain 'line2'")
	}
	if !foundLine3 {
		t.Error("expected diff from first commit to contain 'line3'")
	}
}

func TestIntegrationSingleCommitMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)
	dir := initTestRepo(t)

	commitFile(t, dir, "a.txt", "alpha\n", "initial")
	commitFile(t, dir, "a.txt", "alpha\nbeta\n", "add beta")

	// Single commit mode: "HEAD~1" means show diff of that commit's parent to HEAD~1?
	// Actually in the CLI, single arg is "commit" mode with cfg.Base set to the arg.
	// The diff is then git diff <base> with no target (working tree? No, let's use HEAD~1..HEAD)
	// Let me use "HEAD~1" "HEAD" (compare mode) instead
	baseURL, cleanup := startBinary(t, binPath, dir, "HEAD~1", "HEAD")
	defer cleanup()

	resp, err := http.Get(baseURL + "/api/diff")
	if err != nil {
		t.Fatalf("GET /api/diff: %v", err)
	}
	defer resp.Body.Close()

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected files in diff")
	}
}

func TestIntegrationCLIHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)

	cmd := exec.Command(binPath, "--help")
	out, err := cmd.CombinedOutput()
	// --help should exit 0 (not an error)
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected --help to contain 'Usage:', got:\n%s", output)
	}
	if !strings.Contains(output, "no-open") {
		t.Errorf("expected --help to mention no-open flag, got:\n%s", output)
	}
}

func TestIntegrationInvalidArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)

	tests := []struct {
		name string
		args []string
	}{
		{"too many args", []string{"a", "b", "c"}},
		{"invalid port", []string{"--port", "99999"}},
		{"invalid mode", []string{"--mode", "invalid"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			err := cmd.Run()
			if err == nil {
				t.Error("expected non-zero exit code for invalid args")
			}
		})
	}
}

func TestIntegrationMultipleFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)
	dir := initTestRepo(t)

	commitFile(t, dir, "a.txt", "alpha\n", "add a")
	commitFile(t, dir, "b.txt", "beta\n", "add b")

	// Compare the two commits to see b.txt added
	baseURL, cleanup := startBinary(t, binPath, dir, "HEAD~1", "HEAD")
	defer cleanup()

	resp, err := http.Get(baseURL + "/api/diff")
	if err != nil {
		t.Fatalf("GET /api/diff: %v", err)
	}
	defer resp.Body.Close()

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected at least one file")
	}

	// The diff between HEAD~1 and HEAD should show b.txt as added
	found := false
	for _, f := range result.Files {
		if f.NewName == "b.txt" {
			found = true
			if f.Status != "added" {
				t.Errorf("expected b.txt to be 'added', got %q", f.Status)
			}
		}
	}
	if !found {
		names := make([]string, len(result.Files))
		for i, f := range result.Files {
			names[i] = f.NewName
		}
		t.Errorf("expected b.txt in diff files, got: %v", names)
	}
}

func TestIntegrationCSSAsset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "f.txt", "content\n", "init")

	baseURL, cleanup := startBinary(t, binPath, dir, "HEAD~1", "HEAD")
	defer cleanup()

	// Verify CSS file is served
	resp, err := http.Get(baseURL + "/css/style.css")
	if err != nil {
		t.Fatalf("GET /css/style.css: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for CSS asset, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("CSS file should not be empty")
	}
}

func TestIntegrationJSAsset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "f.txt", "content\n", "init")

	baseURL, cleanup := startBinary(t, binPath, dir, "HEAD~1", "HEAD")
	defer cleanup()

	// Verify JS file is served
	resp, err := http.Get(baseURL + "/js/app.js")
	if err != nil {
		t.Fatalf("GET /js/app.js: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for /js/app.js, got %d", resp.StatusCode)
	}
}

func TestIntegration404(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildBinary(t)

	diffData := fmt.Sprintf("diff --git a/x.txt b/x.txt\n--- a/x.txt\n+++ b/x.txt\n@@ -1 +1 @@\n-a\n+b\n")
	baseURL, cleanup := startBinaryStdin(t, binPath, diffData)
	defer cleanup()

	resp, err := http.Get(baseURL + "/nonexistent-path-xyz")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()

	// FileServer returns 404 for missing files
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 for nonexistent path, got %d", resp.StatusCode)
	}
}
