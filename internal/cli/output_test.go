package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintOutput(t *testing.T) {
	testData := struct {
		Name string `json:"name"`
	}{Name: "test"}

	t.Run("human-readable output", func(t *testing.T) {
		jsonFlag = false
		buf := new(bytes.Buffer)
		err := PrintOutput(buf, "hello human\n", testData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "hello human\n" {
			t.Errorf("expected %q, got %q", "hello human\n", buf.String())
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		jsonFlag = true
		buf := new(bytes.Buffer)
		err := PrintOutput(buf, "hello human\n", testData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed struct {
			Name string `json:"name"`
		}
		err = json.Unmarshal(buf.Bytes(), &parsed)
		if err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}
		if parsed.Name != "test" {
			t.Errorf("expected name to be %q, got %q", "test", parsed.Name)
		}
	})
}
