package cli

import (
	"encoding/json"
	"io"
)

// PrintOutput formats and prints the output to w. If --json is set, it marshals data to JSON.
// Otherwise, it prints humanStr.
func PrintOutput(w io.Writer, humanStr string, data interface{}) error {
	if jsonFlag {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}
	_, err := io.WriteString(w, humanStr)
	return err
}
