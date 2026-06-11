package notes

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditAppendsBodyAndUpdatesTimestamps(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
	})
	require.NoError(t, err)

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	edited, err := Edit(EditInput{
		RootDir:  root,
		Selector: created.Slug,
		Append:   []byte("Next step"),
	})
	require.NoError(t, err)
	assert.NotEqual(t, created.ContentHash, edited.ContentHash)

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, "## Summary\nNext step", string(note.Body))
	assert.Equal(t, "2026-06-02T12:34:56Z", note.CreatedAt.UTC().Format(time.RFC3339))
	assert.Equal(t, "2026-06-02T12:35:56Z", note.UpdatedAt.UTC().Format(time.RFC3339))
}

func TestEditSetsFrontmatterField(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	_, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
	})
	require.NoError(t, err)

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: "auth-migration",
		Set: map[string]string{
			"type": "decision",
		},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, "decision", note.Type)
}

func TestEditRejectsProtectedFrontmatterField(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	_, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	require.NoError(t, err)

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: "auth-migration",
		Set: map[string]string{
			"created_at": "2026-06-02T12:00:00Z",
		},
	})
	require.Error(t, err)
	var appErr *app.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, app.CodeUnsafe, appErr.Code)
}

func TestEditRejectsContentHashMismatch(t *testing.T) {
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

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: created.Slug,
		IfMatch:  "deadbeef",
		Append:   []byte("Next step"),
	})
	require.Error(t, err)
	var appErr *app.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, app.CodeUnsafe, appErr.Code)

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Empty(t, string(note.Body))
}

func TestEditReturnsBusyErrorWhenWriteLockHeld(t *testing.T) {
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

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: created.Slug,
		Append:   []byte("blocked"),
	})
	require.Error(t, err)

	var appErr *app.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, app.CodeUnsafe, appErr.Code)
}

func TestConcurrentEditWithSameHashAllowsOnlyOneSuccess(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
	})
	require.NoError(t, err)

	type outcome struct {
		result EditResult
		err    error
	}

	appends := [][]byte{
		[]byte("First change\n"),
		[]byte("Second change\n"),
	}
	results := make([]outcome, len(appends))

	start := make(chan struct{})
	var wg sync.WaitGroup
	for idx, appendBody := range appends {
		wg.Add(1)
		go func(i int, body []byte) {
			defer wg.Done()
			<-start
			results[i].result, results[i].err = Edit(EditInput{
				RootDir:  root,
				Selector: created.Slug,
				IfMatch:  created.ContentHash,
				Append:   body,
			})
		}(idx, appendBody)
	}

	close(start)
	wg.Wait()

	successes := 0
	preconditionFailures := 0
	for _, result := range results {
		if result.err == nil {
			successes++
			continue
		}
		var appErr *app.AppError
		require.ErrorAs(t, result.err, &appErr)
		assert.Equal(t, app.CodeUnsafe, appErr.Code)
		require.Contains(t, result.err.Error(), "content hash mismatch")
		preconditionFailures++
	}

	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, preconditionFailures)

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)

	body := string(note.Body)
	assert.False(t, strings.Contains(body, "First change\nSecond change\n") || strings.Contains(body, "Second change\nFirst change\n"))
	assert.True(t, body == "## Summary\nFirst change\n" || body == "## Summary\nSecond change\n")
}