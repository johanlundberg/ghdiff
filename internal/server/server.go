// Package server provides the HTTP server for the diff viewer frontend and API.
package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/lundberg/gitdiffview/internal/cli"
	"github.com/lundberg/gitdiffview/internal/diff"
	"github.com/lundberg/gitdiffview/internal/git"
)

// Server is the HTTP server that serves the frontend and API endpoints.
type Server struct {
	config    *cli.Config
	repo      *git.Repo
	mux       *http.ServeMux
	stdinDiff *diff.Result
	assets    fs.FS
	token     string

	indexOnce sync.Once
	indexHTML []byte
}

// New creates a new server. If stdinDiff is non-nil, the server is in stdin mode.
func New(config *cli.Config, repo *git.Repo, stdinDiff *diff.Result, assets fs.FS) *Server {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}

	s := &Server{
		config:    config,
		repo:      repo,
		mux:       http.NewServeMux(),
		stdinDiff: stdinDiff,
		assets:    assets,
		token:     hex.EncodeToString(b),
	}
	s.routes()
	return s
}

// Handler returns the http.Handler (useful for testing).
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/diff", s.requireToken(s.handleDiff))
	s.mux.HandleFunc("GET /api/commits", s.requireToken(s.handleCommits))
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.Handle("GET /", http.FileServerFS(s.assets))
}

// requireToken returns middleware that checks the X-Auth-Token header on API routes.
func (s *Server) requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Auth-Token")), []byte(s.token)) != 1 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// handleIndex serves index.html with the auth token injected.
func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	s.indexOnce.Do(func() {
		raw, err := fs.ReadFile(s.assets, "index.html")
		if err != nil {
			// Will serve an error on every request; acceptable since this is fatal.
			return
		}
		s.indexHTML = []byte(strings.Replace(
			string(raw),
			"{{TOKEN}}",
			s.token,
			1,
		))
	})
	if s.indexHTML == nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(s.indexHTML)
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

func (s *Server) handleCommits(w http.ResponseWriter, _ *http.Request) {
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
