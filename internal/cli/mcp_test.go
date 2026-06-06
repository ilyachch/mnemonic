package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/project"
)

func TestMCPCommandHelpShowsProjectFlag(t *testing.T) {
	setWritableMCPEnv(t)
	res := executeCommand("mcp", "--help")
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}

	if !strings.Contains(res.Stdout, "--project") {
		t.Fatalf("help output does not mention --project: %q", res.Stdout)
	}
}

func TestMCPCommandReturnsClearErrorWithoutProjectContext(t *testing.T) {
	setWritableMCPEnv(t)
	cwd := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	res := executeCommand("mcp")
	if res.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", res.Stdout)
	}
	if res.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(res.Err.Error(), ".mnemonic not found") {
		t.Fatalf("error = %v, want clear missing-project-context error", res.Err)
	}
	if !strings.Contains(res.Stderr, ".mnemonic not found") {
		t.Fatalf("stderr = %q, want clear missing-project-context error", res.Stderr)
	}
}

func TestMCPCommandRejectsBadEnvironmentProjectBeforeServing(t *testing.T) {
	setWritableMCPEnv(t)
	cwd := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(cwd, ".mnemonic"))

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	t.Setenv("MNEMONIC_PROJECT", "missing")

	res := executeCommand("mcp")
	if res.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(res.Err.Error(), "project \"missing\" not found") {
		t.Fatalf("error = %v, want missing-project error", res.Err)
	}
	if res.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", res.Stdout)
	}
}

func writeMCPMnemonicFile(t *testing.T, path string) {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	file := project.NewMnemonicFile()
	file.CreatedAt = now
	file.UpdatedAt = now
	file.Projects = []project.MnemonicProject{
		{
			ID:                    "550e8400-e29b-41d4-a716-446655440000",
			Name:                  "personal",
			Slug:                  "personal",
			Kind:                  project.ProjectKindLocal,
			MemoriesPath:          ".mnemonic-memories/personal",
			MarkdownFormatVersion: 1,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}
	if err := project.WriteMnemonicFile(path, file); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}
}

func setWritableMCPEnv(t *testing.T) {
	t.Helper()

	base := t.TempDir()
	t.Setenv("HOME", base)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))
	t.Setenv("GOCACHE", filepath.Join(base, "go-build"))
	t.Setenv("GOMODCACHE", filepath.Join(base, "go-mod"))
	t.Setenv("GOSUMDB", "off")
}
