package cli

import (
	"errors"

	"github.com/ilyachch/mnemonic/internal/apperr"
)

// ExitCodeForError returns the integer exit code corresponding to the given error.
func ExitCodeForError(err error) int {
	if err == nil {
		return int(apperr.CodeSuccess)
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return int(appErr.Code)
	}

	// Unclassified/standard errors default to internal error code 1.
	return int(apperr.CodeInternal)
}
