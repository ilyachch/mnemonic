package markdown

import "fmt"

type FrontmatterFieldError struct {
	Field string
	Kind  string
	Err   error
}

func (e *FrontmatterFieldError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("frontmatter %q: %s: %v", e.Field, e.Kind, e.Err)
	}
	return fmt.Sprintf("frontmatter %q: %s", e.Field, e.Kind)
}

func (e *FrontmatterFieldError) Unwrap() error {
	return e.Err
}

const (
	FieldErrKindInvalidType       = "invalid_type"
	FieldErrKindInvalidString     = "invalid_string"
	FieldErrKindInvalidStringList = "invalid_string_list"
	FieldErrKindInvalidTimestamp  = "invalid_timestamp"
)

func newStringFieldError(key string, err error) error {
	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidString, Err: err}
}

func newStringSliceFieldError(key string, err error) error {
	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidStringList, Err: err}
}

func newTimeFieldError(key string, err error) error {
	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidTimestamp, Err: err}
}
