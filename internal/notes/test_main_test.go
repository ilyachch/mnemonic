package notes

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	stateHome, err := os.MkdirTemp("", "mnemonic-notes-state-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(stateHome)

	_ = os.Setenv("XDG_STATE_HOME", stateHome)
	_ = os.Setenv("MNEMONIC_STATE_HOME", "")

	os.Exit(m.Run())
}
