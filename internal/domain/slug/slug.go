package slug

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	ErrEmptySlug            = errors.New("slug cannot be empty")
	ErrUnsupportedSlugInput = errors.New("slugify supports ASCII input only")
)

// Slugify converts an ASCII name into a deterministic slug.
func Slugify(input string) (string, error) {
	if input == "" {
		return "", ErrEmptySlug
	}

	var b strings.Builder
	lastWasDash := false

	for _, r := range input {
		if r > unicode.MaxASCII {
			return "", fmt.Errorf("%w: %q", ErrUnsupportedSlugInput, input)
		}

		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteByte(byte(unicode.ToLower(r)))
			lastWasDash = false
		default:
			if b.Len() == 0 || lastWasDash {
				continue
			}
			b.WriteByte('-')
			lastWasDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "", ErrEmptySlug
	}

	return slug, nil
}
