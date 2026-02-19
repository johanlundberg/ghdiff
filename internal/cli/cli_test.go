package cli

import (
	"testing"
)

func TestParseArgs_DefaultConfig(t *testing.T) {
	cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 0 {
		t.Errorf("expected Port=0 (auto), got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected Host=localhost, got %q", cfg.Host)
	}
	if cfg.NoOpen != false {
		t.Errorf("expected NoOpen=false, got %v", cfg.NoOpen)
	}
	if cfg.ViewMode != "split" {
		t.Errorf("expected ViewMode=split, got %q", cfg.ViewMode)
	}
	// No positional args means merge-base mode (requires git, resolved later)
	if cfg.Mode != "merge-base" {
		t.Errorf("expected Mode=merge-base, got %q", cfg.Mode)
	}
}

func TestParseArgs_SingleCommitArg(t *testing.T) {
	cfg, err := ParseArgs([]string{"abc123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "commit" {
		t.Errorf("expected Mode=commit, got %q", cfg.Mode)
	}
	if cfg.Base != "abc123" {
		t.Errorf("expected Base=abc123, got %q", cfg.Base)
	}
}

func TestParseArgs_TwoRefArgs(t *testing.T) {
	cfg, err := ParseArgs([]string{"HEAD~3", "HEAD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "compare" {
		t.Errorf("expected Mode=compare, got %q", cfg.Mode)
	}
	if cfg.Base != "HEAD~3" {
		t.Errorf("expected Base=HEAD~3, got %q", cfg.Base)
	}
	if cfg.Target != "HEAD" {
		t.Errorf("expected Target=HEAD, got %q", cfg.Target)
	}
}

func TestParseArgs_StdinDash(t *testing.T) {
	cfg, err := ParseArgs([]string{"-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "stdin" {
		t.Errorf("expected Mode=stdin, got %q", cfg.Mode)
	}
}

func TestParseArgs_PortFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"--port", "8080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port=8080, got %d", cfg.Port)
	}
}

func TestParseArgs_HostFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"--host", "0.0.0.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host=0.0.0.0, got %q", cfg.Host)
	}
}

func TestParseArgs_NoOpenFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"--no-open"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NoOpen != true {
		t.Errorf("expected NoOpen=true, got %v", cfg.NoOpen)
	}
}

func TestParseArgs_ModeFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"--mode", "unified"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ViewMode != "unified" {
		t.Errorf("expected ViewMode=unified, got %q", cfg.ViewMode)
	}
}

func TestParseArgs_ModeFlagSplit(t *testing.T) {
	cfg, err := ParseArgs([]string{"--mode", "split"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ViewMode != "split" {
		t.Errorf("expected ViewMode=split, got %q", cfg.ViewMode)
	}
}

func TestParseArgs_InvalidModeFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--mode", "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}
}

func TestParseArgs_TooManyArgs(t *testing.T) {
	_, err := ParseArgs([]string{"a", "b", "c"})
	if err == nil {
		t.Fatal("expected error for too many args, got nil")
	}
}

func TestParseArgs_FlagsWithPositionalArgs(t *testing.T) {
	cfg, err := ParseArgs([]string{"--port", "3000", "--no-open", "abc123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "commit" {
		t.Errorf("expected Mode=commit, got %q", cfg.Mode)
	}
	if cfg.Base != "abc123" {
		t.Errorf("expected Base=abc123, got %q", cfg.Base)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected Port=3000, got %d", cfg.Port)
	}
	if cfg.NoOpen != true {
		t.Errorf("expected NoOpen=true, got %v", cfg.NoOpen)
	}
}

func TestParseArgs_StdinWithFlags(t *testing.T) {
	cfg, err := ParseArgs([]string{"--port", "9090", "-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "stdin" {
		t.Errorf("expected Mode=stdin, got %q", cfg.Mode)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected Port=9090, got %d", cfg.Port)
	}
}

func TestParseArgs_TwoRefsWithFlags(t *testing.T) {
	cfg, err := ParseArgs([]string{"--mode", "unified", "HEAD~5", "HEAD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "compare" {
		t.Errorf("expected Mode=compare, got %q", cfg.Mode)
	}
	if cfg.ViewMode != "unified" {
		t.Errorf("expected ViewMode=unified, got %q", cfg.ViewMode)
	}
	if cfg.Base != "HEAD~5" {
		t.Errorf("expected Base=HEAD~5, got %q", cfg.Base)
	}
	if cfg.Target != "HEAD" {
		t.Errorf("expected Target=HEAD, got %q", cfg.Target)
	}
}

func TestParseArgs_WorkingDot(t *testing.T) {
	cfg, err := ParseArgs([]string{"."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "working" {
		t.Errorf("expected Mode=working, got %q", cfg.Mode)
	}
}

func TestParseArgs_InvalidPortNegative(t *testing.T) {
	_, err := ParseArgs([]string{"--port", "-1"})
	if err == nil {
		t.Fatal("expected error for negative port, got nil")
	}
}

func TestParseArgs_InvalidPortTooHigh(t *testing.T) {
	_, err := ParseArgs([]string{"--port", "99999"})
	if err == nil {
		t.Fatal("expected error for port > 65535, got nil")
	}
}

func TestParseArgs_HelpFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--help"})
	if err != ErrHelp {
		t.Errorf("expected ErrHelp, got %v", err)
	}
}
