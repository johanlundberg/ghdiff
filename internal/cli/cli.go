// Package cli handles command-line argument parsing and configuration.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// ErrHelp is returned when --help is requested.
var ErrHelp = errors.New("help requested")

// Config holds the parsed CLI configuration.
type Config struct {
	Mode     string // "merge-base", "commit", "compare", "working", "stdin"
	Base     string // base ref for diff
	Target   string // target ref (or empty for working tree)
	Port     int
	Host     string
	NoOpen   bool
	ViewMode string // "split" or "unified"
}

const usageHeader = `Usage: gitdiffview [flags] [ref1 [ref2]]

Display git diffs in a GitHub-style web UI.

Arguments:
  (none)         diff working tree against merge-base with main/master
  <commit>       show diff for a single commit
  <ref1> <ref2>  diff between two refs
  -              read unified diff from stdin

Flags:
`

// flags holds pointers to flag values, used to share between
// newFlagSet and ParseArgs without duplicating definitions.
type flags struct {
	port     int
	host     string
	noOpen   bool
	viewMode string
}

func newFlagSet(f *flags) *flag.FlagSet {
	fs := flag.NewFlagSet("gitdiffview", flag.ContinueOnError)
	fs.IntVar(&f.port, "port", 0, "HTTP server port (0 = auto)")
	fs.StringVar(&f.host, "host", "localhost", "HTTP server host")
	fs.BoolVar(&f.noOpen, "no-open", false, "don't open browser automatically")
	fs.StringVar(&f.viewMode, "mode", "split", "view mode: split or unified")
	return fs
}

// ParseArgs parses command-line arguments into a Config.
// It does not execute git commands; mode="merge-base" signals
// that the caller must resolve the actual merge-base ref.
func ParseArgs(args []string) (*Config, error) {
	var f flags
	fs := newFlagSet(&f)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, err
	}

	// Validate view mode
	if f.viewMode != "split" && f.viewMode != "unified" {
		return nil, fmt.Errorf("invalid mode %q: must be split or unified", f.viewMode)
	}

	// Validate port range
	if f.port < 0 || f.port > 65535 {
		return nil, fmt.Errorf("invalid port: %d (must be 0-65535)", f.port)
	}

	cfg := &Config{
		Port:     f.port,
		Host:     f.host,
		NoOpen:   f.noOpen,
		ViewMode: f.viewMode,
	}

	positional := fs.Args()
	switch len(positional) {
	case 0:
		cfg.Mode = "merge-base"
	case 1:
		switch positional[0] {
		case "-":
			cfg.Mode = "stdin"
		case ".":
			cfg.Mode = "working"
		default:
			cfg.Mode = "commit"
			cfg.Base = positional[0]
		}
	case 2:
		cfg.Mode = "compare"
		cfg.Base = positional[0]
		cfg.Target = positional[1]
	default:
		return nil, fmt.Errorf("too many arguments: expected at most 2, got %d", len(positional))
	}

	return cfg, nil
}

// PrintUsage writes usage information to w.
func PrintUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageHeader)
	var f flags
	fs := newFlagSet(&f)
	fs.SetOutput(w)
	fs.PrintDefaults()
}
