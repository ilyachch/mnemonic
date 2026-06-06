package cli

import (
	"testing"
)

func TestHelloCmd(t *testing.T) {
	res := executeCommand("hello")
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}

	expected := "hello world\n"
	if res.Stdout != expected {
		t.Errorf("expected %q, got %q", expected, res.Stdout)
	}

	if res.Stderr != "" {
		t.Errorf("expected empty stderr, got %q", res.Stderr)
	}
}
