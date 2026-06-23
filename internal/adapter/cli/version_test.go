package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ilyachch/mnemonic/internal/platform/buildinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	res := executeCommand("version")
	require.NoError(t, res.Err)

	assert.Equal(t, "dev", strings.TrimSpace(res.Stdout))
	assert.Empty(t, res.Stderr)
}

func TestVersionCmd_JSON(t *testing.T) {
	res := executeCommand("version", "--json")
	require.NoError(t, res.Err)

	var parsed struct {
		Version string `json:"version"`
	}
	err := json.Unmarshal([]byte(res.Stdout), &parsed)
	require.NoError(t, err, "stdout: %s", res.Stdout)

	assert.Equal(t, buildinfo.Version(), parsed.Version)
	assert.Empty(t, res.Stderr)
}
