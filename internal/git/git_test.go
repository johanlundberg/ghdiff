package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repo with user config and an initial commit.
// Returns the path to the repo directory.
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

func TestGetMainBranch_Main(t *testing.T) {
	dir := initTestRepo(t)
	// Modern git defaults to "main" or "master" depending on config.
	// Force a "main" branch by renaming.
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rename branch: %v\n%s", err, out)
	}
	commitFile(t, dir, "README.md", "hello", "initial commit")

	repo := NewRepo(dir)
	branch, err := repo.GetMainBranch()
	if err != nil {
		t.Fatalf("GetMainBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got %q", branch)
	}
}

func TestGetMainBranch_Master(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "master")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rename branch: %v\n%s", err, out)
	}
	commitFile(t, dir, "README.md", "hello", "initial commit")

	repo := NewRepo(dir)
	branch, err := repo.GetMainBranch()
	if err != nil {
		t.Fatalf("GetMainBranch: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected 'master', got %q", branch)
	}
}

func TestGetMainBranch_Neither(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "develop")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rename branch: %v\n%s", err, out)
	}
	commitFile(t, dir, "README.md", "hello", "initial commit")

	repo := NewRepo(dir)
	_, err = repo.GetMainBranch()
	if err == nil {
		t.Error("expected error when neither main nor master exists")
	}
}

func TestGetMergeBase(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	// Create initial commit on main
	baseHash := commitFile(t, dir, "README.md", "hello", "initial commit")

	// Create a feature branch from this point
	for _, args := range [][]string{
		{"git", "checkout", "-b", "feature"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup branch %v: %v\n%s", args, err, out)
		}
	}

	// Add a commit on feature branch
	commitFile(t, dir, "feature.txt", "feature work", "feature commit")

	// Switch back to main and add a commit
	cmd = exec.Command("git", "checkout", "main")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("checkout main: %v\n%s", err, out)
	}
	commitFile(t, dir, "main.txt", "main work", "main commit")

	repo := NewRepo(dir)
	mergeBase, err := repo.GetMergeBase("main", "feature")
	if err != nil {
		t.Fatalf("GetMergeBase: %v", err)
	}
	if mergeBase != baseHash {
		t.Errorf("expected merge-base %q, got %q", baseHash, mergeBase)
	}
}

func TestGetDiff_BetweenRefs(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	commitFile(t, dir, "file.txt", "line1\n", "first commit")

	// Create second commit with changed content
	commitFile(t, dir, "file.txt", "line1\nline2\n", "second commit")

	repo := NewRepo(dir)
	diff, err := repo.GetDiff("HEAD~1", "HEAD")
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if !strings.Contains(diff, "+line2") {
		t.Errorf("expected diff to contain '+line2', got:\n%s", diff)
	}
	if !strings.Contains(diff, "file.txt") {
		t.Errorf("expected diff to reference 'file.txt', got:\n%s", diff)
	}
}

func TestGetDiff_WorkingTree(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	commitFile(t, dir, "file.txt", "original\n", "initial commit")

	// Modify the file without committing (unstaged change)
	err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("modified\n"), 0644)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}

	repo := NewRepo(dir)
	diff, err := repo.GetDiff("HEAD", "")
	if err != nil {
		t.Fatalf("GetDiff working tree: %v", err)
	}
	if !strings.Contains(diff, "-original") {
		t.Errorf("expected diff to contain '-original', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+modified") {
		t.Errorf("expected diff to contain '+modified', got:\n%s", diff)
	}
}

func TestGetCommits(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	commitFile(t, dir, "a.txt", "a", "first commit")
	commitFile(t, dir, "b.txt", "b", "second commit")
	commitFile(t, dir, "c.txt", "c", "third commit")

	repo := NewRepo(dir)
	commits, err := repo.GetCommits(2)
	if err != nil {
		t.Fatalf("GetCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	// Most recent commit first
	if commits[0].Message != "third commit" {
		t.Errorf("expected first commit message 'third commit', got %q", commits[0].Message)
	}
	if commits[1].Message != "second commit" {
		t.Errorf("expected second commit message 'second commit', got %q", commits[1].Message)
	}
	// Verify fields are populated
	for i, c := range commits {
		if c.Hash == "" {
			t.Errorf("commit %d: empty hash", i)
		}
		if c.Author != "Test User" {
			t.Errorf("commit %d: expected author 'Test User', got %q", i, c.Author)
		}
		if c.Date == "" {
			t.Errorf("commit %d: empty date", i)
		}
	}
}

func TestGetCommits_All(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	cmd.CombinedOutput()

	commitFile(t, dir, "a.txt", "a", "first commit")
	commitFile(t, dir, "b.txt", "b", "second commit")

	repo := NewRepo(dir)
	// Request more commits than exist
	commits, err := repo.GetCommits(10)
	if err != nil {
		t.Fatalf("GetCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits (all available), got %d", len(commits))
	}
}

func TestGetDiff_RejectsFlagLikeRef(t *testing.T) {
	repo := NewRepo(".")

	tests := []struct {
		name   string
		base   string
		target string
	}{
		{"base starts with dash", "--output=/tmp/evil", "HEAD"},
		{"base is flag", "-n", "HEAD"},
		{"target starts with dash", "HEAD", "--output=/tmp/evil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.GetDiff(tt.base, tt.target)
			if err == nil {
				t.Error("expected error for flag-like ref, got nil")
			}
			if !strings.Contains(err.Error(), "must not start with '-'") {
				t.Errorf("expected error about '-' prefix, got: %v", err)
			}
		})
	}
}
