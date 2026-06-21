package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteDryRunReturnsTrashPath(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	require.NoError(t, err)

	result, err := Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
		DryRun:   true,
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "trash", result.Mode)
	assert.NotEmpty(t, result.TrashPath)

	_, err = os.Stat(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	_, err = os.Stat(result.TrashPath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteMovesNoteToTrash(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("body\n"),
	})
	require.NoError(t, err)

	result, err := Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
	})
	require.NoError(t, err)
	assert.Equal(t, "trash", result.Mode)
	assert.NotEmpty(t, result.TrashPath)

	_, err = os.Stat(filepath.Join(root, "auth-migration.md"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(result.TrashPath)
	require.NoError(t, err)

	list, err := List(root)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestDeleteHardRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("body\n"),
	})
	require.NoError(t, err)

	_, err = Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
		Hard:     true,
	})
	require.Error(t, err)
}

func TestDeleteHardRemovesFileWhenConfirmed(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("body\n"),
	})
	require.NoError(t, err)

	result, err := Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
		Hard:     true,
		Yes:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "hard", result.Mode)

	_, err = os.Stat(filepath.Join(root, "auth-migration.md"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root, ".trash"))
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteRejectsMissingSelector(t *testing.T) {
	_, err := Delete(DeleteInput{})
	require.Error(t, err)
}

func TestDeleteReturnsBusyErrorWhenWriteLockHeld(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	require.NoError(t, err)

	guard, err := acquireWriteLock(root)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, guard.Release())
	}()

	_, err = Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
	})
	require.Error(t, err)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeUnsafe, appErr.Code)
}
