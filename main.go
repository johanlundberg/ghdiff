// ghdiff displays git diffs in a GitHub-style web UI.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/lundberg/ghdiff/internal/browser"
	"github.com/lundberg/ghdiff/internal/cli"
	"github.com/lundberg/ghdiff/internal/diff"
	"github.com/lundberg/ghdiff/internal/git"
	"github.com/lundberg/ghdiff/internal/server"
	"github.com/lundberg/ghdiff/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, cli.ErrHelp) {
			cli.PrintUsage(os.Stderr)
			return nil
		}
		return err
	}

	repo := git.NewRepo(".")
	var stdinDiff *diff.Result

	switch cfg.Mode {
	case "stdin":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		result, err := diff.Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing diff from stdin: %w", err)
		}
		stdinDiff = result

	case "merge-base":
		mainBranch, err := repo.GetMainBranch()
		if err != nil {
			return fmt.Errorf("detecting main branch: %w", err)
		}
		base, err := repo.GetMergeBase("HEAD", mainBranch)
		if err != nil {
			return fmt.Errorf("computing merge-base: %w", err)
		}
		cfg.Base = base

	case "working":
		cfg.Base = "HEAD"

	case "commit", "compare":
		// Base (and Target for compare) already set by CLI parser
	}

	// Listen on a port to get the actual address (handles port=0 auto-select)
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	// Extract the actual port (important when port=0).
	// The type assertion is safe because net.Listen("tcp", ...) always returns *net.TCPAddr.
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("unexpected listener address type: %T", ln.Addr())
	}
	actualPort := tcpAddr.Port
	cfg.Port = actualPort
	url := fmt.Sprintf("http://%s", net.JoinHostPort(cfg.Host, strconv.Itoa(actualPort)))

	fmt.Printf("Listening on %s\n", url)
	if cfg.Host != "localhost" && cfg.Host != "127.0.0.1" {
		fmt.Fprintln(os.Stderr, "WARNING: ghdiff is not designed for public access. It exposes repository contents without authentication.")
	}
	fmt.Println("Press Ctrl+C to stop")

	if !cfg.NoOpen {
		if err := browser.Open(url); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not open browser: %v\n", err)
		}
	}

	srv := server.New(cfg, repo, stdinDiff, web.Assets)
	httpServer := &http.Server{Handler: srv.Handler()}

	// Graceful shutdown on Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		fmt.Println("\nShutting down...")
		_ = httpServer.Close()
	}()

	if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
