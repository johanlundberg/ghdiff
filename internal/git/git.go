package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Commit represents a single git commit.
type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// Repo represents a git repository at a specific directory.
type Repo struct {
	Dir string
}

// NewRepo creates a Repo pointing at the given directory.
func NewRepo(dir string) *Repo {
	return &Repo{Dir: dir}
}

// git runs a git command in the repo directory and returns trimmed stdout.
func (r *Repo) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetMainBranch returns "main" or "master", whichever exists as a local branch.
func (r *Repo) GetMainBranch() (string, error) {
	// Check if "main" branch exists
	if _, err := r.git("rev-parse", "--verify", "refs/heads/main"); err == nil {
		return "main", nil
	}
	// Check if "master" branch exists
	if _, err := r.git("rev-parse", "--verify", "refs/heads/master"); err == nil {
		return "master", nil
	}
	return "", fmt.Errorf("neither 'main' nor 'master' branch found")
}

// GetMergeBase returns the merge-base commit hash between two refs.
func (r *Repo) GetMergeBase(ref1, ref2 string) (string, error) {
	return r.git("merge-base", ref1, ref2)
}

// GetDiff returns unified diff text between two refs.
// If target is empty, diffs base against the working tree (staged + unstaged).
func (r *Repo) GetDiff(base, target string) (string, error) {
	if target == "" {
		return r.git("diff", "--no-ext-diff", base)
	}
	return r.git("diff", "--no-ext-diff", base, target)
}

// GetCommits returns the most recent n commits for the current branch.
func (r *Repo) GetCommits(n int) ([]Commit, error) {
	// Use a separator unlikely to appear in commit messages
	sep := "---COMMIT_SEP---"
	format := strings.Join([]string{"%H", "%s", "%an", "%ai"}, sep)
	out, err := r.git("log", "--format="+format, "-n", strconv.Itoa(n))
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, sep, 4)
		if len(parts) != 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return commits, nil
}
