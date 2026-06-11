package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintOutput(t *testing.T) {
	testData := struct {
		Name string `json:"name"`
	}{Name: "test"}

	t.Run("human-readable output", func(t *testing.T) {
		jsonFlag = false
		buf := new(bytes.Buffer)
		err := PrintOutput(buf, "hello human\n", testData)
		require.NoError(t, err)
		assert.Equal(t, "hello human\n", buf.String())
	})

	t.Run("JSON output", func(t *testing.T) {
		jsonFlag = true
		buf := new(bytes.Buffer)
		err := PrintOutput(buf, "hello human\n", testData)
		require.NoError(t, err)

		var parsed struct {
			Name string `json:"name"`
		}
		err = json.Unmarshal(buf.Bytes(), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "test", parsed.Name)
	})
}