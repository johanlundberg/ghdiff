package server

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/lundberg/gitdiffview/internal/cli"
	"github.com/lundberg/gitdiffview/internal/diff"
	"github.com/lundberg/gitdiffview/internal/git"
)

// Server is the HTTP server that serves the frontend and API endpoints.
type Server struct {
	config    *cli.Config
	repo      *git.Repo
	mux       *http.ServeMux
	stdinDiff *diff.DiffResult
	assets    fs.FS
}

// New creates a new server. If stdinDiff is non-nil, the server is in stdin mode.
func New(config *cli.Config, repo *git.Repo, stdinDiff *diff.DiffResult, assets fs.FS) *Server {
	s := &Server{
		config:    config,
		repo:      repo,
		mux:       http.NewServeMux(),
		stdinDiff: stdinDiff,
		assets:    assets,
	}
	s.routes()
	return s
}

// Handler returns the http.Handler (useful for testing).
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/diff", s.handleDiff)
	s.mux.HandleFunc("GET /api/commits", s.handleCommits)
	s.mux.Handle("GET /", http.FileServerFS(s.assets))
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	// In stdin mode, always return the pre-parsed diff
	if s.stdinDiff != nil {
		writeJSON(w, s.stdinDiff)
		return
	}

	// Determine which base ref to use
	base := r.URL.Query().Get("base")
	if base == "" {
		base = s.config.Base
	}

	// Determine which target ref to use
	target := r.URL.Query().Get("target")
	if target == "" {
		target = s.config.Target
	}

	// Get the diff from git
	rawDiff, err := s.repo.GetDiff(base, target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := diff.Parse(rawDiff)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleCommits(w http.ResponseWriter, r *http.Request) {
	// In stdin mode, return empty array
	if s.stdinDiff != nil {
		writeJSON(w, []git.Commit{})
		return
	}

	commits, err := s.repo.GetCommits(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if commits == nil {
		commits = []git.Commit{}
	}

	writeJSON(w, commits)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
