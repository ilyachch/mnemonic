package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestPathUsesStateHomeAndProjectID(t *testing.T) {
	stateHome := filepath.Join(testutil.CleanEnvForTest(t), "state")

	got, err := Path("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	want := filepath.Join(stateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if filepath.Clean(filepath.Dir(got)) == filepath.Clean(filepath.Join(os.Getenv("HOME"), ".mnemonic")) {
		t.Fatal("Path() should not resolve under ~/.mnemonic")
	}
}

func TestPathRejectsEmptyProjectID(t *testing.T) {
	if _, err := Path(""); err == nil {
		t.Fatal("Path() error = nil, want error")
	}
}
