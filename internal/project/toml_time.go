package project

import (
	"time"
)

type tomlTime time.Time

func newTOMLTime(ts time.Time) tomlTime {
	return tomlTime(ts.UTC())
}

func (t tomlTime) Time() time.Time {
	return time.Time(t)
}

func (t tomlTime) MarshalText() ([]byte, error) {
	return []byte(time.Time(t).UTC().Format(time.RFC3339)), nil
}

func (t *tomlTime) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*t = tomlTime{}
		return nil
	}

	ts, err := time.Parse(time.RFC3339, string(text))
	if err != nil {
		return err
	}
	*t = tomlTime(ts.UTC())
	return nil
}
