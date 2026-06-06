package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	res := executeCommand("version")
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}

	actual := strings.TrimSpace(res.Stdout)
	if actual == "" {
		t.Error("expected non-empty version output")
	}

	if res.Stderr != "" {
		t.Errorf("expected empty stderr, got %q", res.Stderr)
	}
}

func TestVersionCmd_JSON(t *testing.T) {
	res := executeCommand("version", "--json")
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}

	var parsed struct {
		Version string `json:"version"`
	}
	err := json.Unmarshal([]byte(res.Stdout), &parsed)
	if err != nil {
		t.Fatalf("failed to parse JSON from stdout %q: %v", res.Stdout, err)
	}

	if parsed.Version != Version {
		t.Errorf("expected version %q, got %q", Version, parsed.Version)
	}

	if res.Stderr != "" {
		t.Errorf("expected empty stderr, got %q", res.Stderr)
	}
}
