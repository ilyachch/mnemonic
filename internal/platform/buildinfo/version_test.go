package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_Default(t *testing.T) {
	// Reset to default state
	version = ""

	got := Version()
	assert.Equal(t, "dev", got)
}

func TestVersion_AfterSet(t *testing.T) {
	v := "1.2.3"
	SetVersion(v)

	got := Version()
	assert.Equal(t, v, got)

	// Clean up
	version = ""
}

func TestVersion_AfterSetEmpty(t *testing.T) {
	SetVersion("")

	got := Version()
	assert.Equal(t, "dev", got)
}
