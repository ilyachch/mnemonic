package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWritesCanonicalNote(t *testing.T) {
	root := t.TempDir()

	got, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	require.NoError(t, err)
	assert.Equal(t, "auth-migration", got.Slug)
	assert.Equal(t, "auth-migration.md", got.Path)
	assert.NotEmpty(t, got.NoteID)
	assert.NotEmpty(t, got.ContentHash)

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	parsed, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, "Auth migration", parsed.Title)
	assert.Equal(t, "auth-migration", parsed.Slug)
}

func TestCreateUsesInjectedClockAndUUID(t *testing.T) {
	root := t.TempDir()
	now := func() time.Time {
		return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	}
	uuidFn := func() string {
		return "550e8400-e29b-41d4-a716-446655440000"
	}

	got, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
		Now:     now,
		UUID:    uuidFn,
	})
	require.NoError(t, err)

	want := CreateResult{
		NoteID:      "550e8400-e29b-41d4-a716-446655440000",
		Slug:        "auth-migration",
		Path:        "auth-migration.md",
		ContentHash: got.ContentHash,
	}
	assert.Equal(t, want.NoteID, got.NoteID)
	assert.Equal(t, want.Slug, got.Slug)
	assert.Equal(t, want.Path, got.Path)

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	parsed, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, want.NoteID, parsed.MnemonicNoteID)
	assert.True(t, parsed.CreatedAt.Equal(now()), "CreatedAt = %s, want %s", parsed.CreatedAt, now())
	assert.True(t, parsed.UpdatedAt.Equal(now()), "UpdatedAt = %s, want %s", parsed.UpdatedAt, now())
}

func TestCreateRejectsDuplicateSlug(t *testing.T) {
	root := t.TempDir()
	_, err := Create(CreateInput{RootDir: root, Title: "Auth migration"})
	require.NoError(t, err)
	_, err = Create(CreateInput{RootDir: root, Title: "Auth migration"})
	require.Error(t, err)
}

func TestCreateRespectsProjectClock(t *testing.T) {
	root := t.TempDir()
	restore := project.SetClock(testClock{})
	defer restore()

	got, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got.NoteID)
}

func TestCreateReturnsBusyErrorWhenWriteLockHeld(t *testing.T) {
	root := t.TempDir()

	guard, err := acquireWriteLock(root)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, guard.Release())
	}()

	_, err = Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	require.Error(t, err)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeUnsafe, appErr.Code)
}

type testClock struct{}

func (testClock) Now() time.Time {
	return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
}

func (testClock) UUID() string {
	return "550e8400-e29b-41d4-a716-446655440000"
}
