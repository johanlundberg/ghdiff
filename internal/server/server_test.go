package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lundberg/gitdiffview/internal/cli"
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

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html><body>Hello diffweb</body></html>"),
		},
	}
}

func TestAPIDiff(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	commitFile(t, dir, "file.txt", "line1\n", "first commit")
	commitFile(t, dir, "file.txt", "line1\nline2\n", "second commit")

	cfg := &cli.Config{
		Mode: "commit",
		Base: "HEAD~1",
		Host: "localhost",
		Port: 0,
	}
	repo := git.NewRepo(dir)
	srv := New(cfg, repo, nil, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/diff")
	if err != nil {
		t.Fatalf("GET /api/diff: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected at least one file in diff result")
	}
	if result.Files[0].NewName != "file.txt" {
		t.Errorf("expected file name 'file.txt', got %q", result.Files[0].NewName)
	}
}

func TestAPIDiffWithBase(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	firstHash := commitFile(t, dir, "file.txt", "line1\n", "first commit")
	commitFile(t, dir, "file.txt", "line1\nline2\n", "second commit")
	commitFile(t, dir, "file.txt", "line1\nline2\nline3\n", "third commit")

	cfg := &cli.Config{
		Mode: "commit",
		Base: "HEAD~1",
		Host: "localhost",
		Port: 0,
	}
	repo := git.NewRepo(dir)
	srv := New(cfg, repo, nil, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Use ?base= to override the config's default base
	resp, err := http.Get(ts.URL + "/api/diff?base=" + firstHash)
	if err != nil {
		t.Fatalf("GET /api/diff?base=...: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected at least one file in diff result")
	}

	// Diffing from first commit should show more changes (line2 and line3 added)
	// The diff should contain hunks with the added lines
	found := false
	for _, f := range result.Files {
		for _, h := range f.Hunks {
			for _, l := range h.Lines {
				if l.Type == "add" && l.Content == "line3" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected diff from first commit to contain added line 'line3'")
	}
}

func TestAPIDiffWithTarget(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	firstHash := commitFile(t, dir, "file.txt", "line1\n", "first commit")
	secondHash := commitFile(t, dir, "file.txt", "line1\nline2\n", "second commit")
	commitFile(t, dir, "file.txt", "line1\nline2\nline3\n", "third commit")

	// Config has Base=first commit, Target="" (working tree).
	// We'll use ?target= to override to a specific commit.
	cfg := &cli.Config{
		Mode: "commit",
		Base: firstHash,
		Host: "localhost",
		Port: 0,
	}
	repo := git.NewRepo(dir)
	srv := New(cfg, repo, nil, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Use ?target= to diff from first commit to second commit only
	resp, err := http.Get(ts.URL + "/api/diff?target=" + secondHash)
	if err != nil {
		t.Fatalf("GET /api/diff?target=...: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected at least one file in diff result")
	}

	// Diffing first..second should show line2 added but NOT line3
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
		t.Error("expected diff first..second to contain added line 'line2'")
	}
	if foundLine3 {
		t.Error("expected diff first..second to NOT contain added line 'line3'")
	}
}

func TestAPIDiffStdinMode(t *testing.T) {
	stdinDiff := &diff.DiffResult{
		Files: []diff.FileDiff{
			{
				OldName: "stdin-file.txt",
				NewName: "stdin-file.txt",
				Status:  "modified",
				Hunks: []diff.Hunk{
					{
						OldStart: 1,
						OldLines: 1,
						NewStart: 1,
						NewLines: 1,
						Lines: []diff.Line{
							{Type: "delete", Content: "old", OldNum: 1},
							{Type: "add", Content: "new", NewNum: 1},
						},
					},
				},
			},
		},
	}

	cfg := &cli.Config{
		Mode: "stdin",
		Host: "localhost",
		Port: 0,
	}
	srv := New(cfg, nil, stdinDiff, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/diff")
	if err != nil {
		t.Fatalf("GET /api/diff: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	if result.Files[0].NewName != "stdin-file.txt" {
		t.Errorf("expected file name 'stdin-file.txt', got %q", result.Files[0].NewName)
	}
}

func TestAPIDiffStdinModeIgnoresBase(t *testing.T) {
	stdinDiff := &diff.DiffResult{
		Files: []diff.FileDiff{
			{NewName: "stdin.txt", Status: "modified"},
		},
	}

	cfg := &cli.Config{
		Mode: "stdin",
		Host: "localhost",
		Port: 0,
	}
	srv := New(cfg, nil, stdinDiff, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Even with ?base= param, stdin mode should return pre-parsed diff
	resp, err := http.Get(ts.URL + "/api/diff?base=abc123")
	if err != nil {
		t.Fatalf("GET /api/diff?base=abc123: %v", err)
	}
	defer resp.Body.Close()

	var result diff.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	if result.Files[0].NewName != "stdin.txt" {
		t.Errorf("expected 'stdin.txt', got %q", result.Files[0].NewName)
	}
}

func TestAPICommits(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	commitFile(t, dir, "a.txt", "a", "first commit")
	commitFile(t, dir, "b.txt", "b", "second commit")

	cfg := &cli.Config{
		Mode: "merge-base",
		Host: "localhost",
		Port: 0,
	}
	repo := git.NewRepo(dir)
	srv := New(cfg, repo, nil, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/commits")
	if err != nil {
		t.Fatalf("GET /api/commits: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var commits []git.Commit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	// Most recent first
	if commits[0].Message != "second commit" {
		t.Errorf("expected first commit message 'second commit', got %q", commits[0].Message)
	}
	if commits[1].Message != "first commit" {
		t.Errorf("expected second commit message 'first commit', got %q", commits[1].Message)
	}
	for i, c := range commits {
		if c.Hash == "" {
			t.Errorf("commit %d: empty hash", i)
		}
		if c.Author == "" {
			t.Errorf("commit %d: empty author", i)
		}
		if c.Date == "" {
			t.Errorf("commit %d: empty date", i)
		}
	}
}

func TestAPICommitsStdinMode(t *testing.T) {
	stdinDiff := &diff.DiffResult{
		Files: []diff.FileDiff{},
	}

	cfg := &cli.Config{
		Mode: "stdin",
		Host: "localhost",
		Port: 0,
	}
	srv := New(cfg, nil, stdinDiff, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/commits")
	if err != nil {
		t.Fatalf("GET /api/commits: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var commits []git.Commit
	if err := json.Unmarshal(body, &commits); err != nil {
		t.Fatalf("decode JSON: %v\nbody: %s", err, body)
	}
	if len(commits) != 0 {
		t.Errorf("expected empty commits array in stdin mode, got %d", len(commits))
	}
}

func TestStaticServing(t *testing.T) {
	cfg := &cli.Config{
		Mode: "stdin",
		Host: "localhost",
		Port: 0,
	}
	stdinDiff := &diff.DiffResult{Files: []diff.FileDiff{}}

	srv := New(cfg, nil, stdinDiff, testAssets())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if !strings.Contains(string(body), "Hello diffweb") {
		t.Errorf("expected body to contain 'Hello diffweb', got:\n%s", body)
	}
}
