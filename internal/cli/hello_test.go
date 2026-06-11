package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelloCmd(t *testing.T) {
	res := executeCommand("hello")
	require.NoError(t, res.Err)

	assert.Equal(t, "hello world\n", res.Stdout)
	assert.Empty(t, res.Stderr)
}