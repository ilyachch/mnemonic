package sqliteindex

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/stretchr/testify/require"
)

func TestScanNotesReturnsDocsAndErrorsInPathOrder(t *testing.T) {
	root := t.TempDir()

	writeScanTestNote(t, filepath.Join(root, "b.md"), "b-id", "Bravo", "bravo", time.Date(2026, time.June, 2, 12, 0, 0, 0, time.UTC))
	writeScanTestNote(t, filepath.Join(root, "a.md"), "a-id", "Alpha", "alpha", time.Date(2026, time.June, 2, 12, 0, 0, 0, time.UTC))
	require.NoError(t, os.WriteFile(filepath.Join(root, "c.md"), []byte("---\nslug: broken\n"), 0o644))

	docs, errs, err := ScanNotes(root)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	require.Len(t, docs, 2)
	require.Equal(t, "a.md", docs[0].RelPath)
	require.Equal(t, "b.md", docs[1].RelPath)
	require.Contains(t, errs[0].Error(), "c.md")
}

func writeScanTestNote(t testing.TB, path, noteID, title, slug string, now time.Time) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	rendered, err := markdown.RenderNote(markdown.Note{
		MnemonicNoteID: noteID,
		Title:          title,
		Slug:           slug,
		CreatedAt:      now,
		UpdatedAt:      now,
		Body:           []byte("# " + title + "\n"),
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, rendered, 0o644))
}

var benchmarkScanDocs []NoteDoc
var benchmarkScanErrs []error

func BenchmarkScanNotesSequentialVsParallel(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.StopTimer()
			root := b.TempDir()
			seedScanBenchmarkVault(b, root, size)
			b.StartTimer()

			b.Run("sequential", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					docs, errs, err := scanNotesSequential(root)
					require.NoError(b, err)
					benchmarkScanDocs = docs
					benchmarkScanErrs = errs
				}
			})

			b.Run("parallel", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					docs, errs, err := ScanNotes(root)
					require.NoError(b, err)
					benchmarkScanDocs = docs
					benchmarkScanErrs = errs
				}
			})
		})
	}
}

func scanNotesSequential(root string) ([]NoteDoc, []error, error) {
	paths, err := (markdownstore.Store{RootDir: root}).Walk()
	if err != nil {
		return nil, nil, err
	}
	docs := make([]NoteDoc, 0, len(paths))
	errs := make([]error, 0)
	for _, relPath := range paths {
		doc, scanErr := scanOneNote(root, relPath)
		if scanErr != nil {
			errs = append(errs, scanErr)
			continue
		}
		docs = append(docs, doc)
	}
	return docs, errs, nil
}

func seedScanBenchmarkVault(b testing.TB, root string, size int) {
	b.Helper()

	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.UTC)
	for i := 0; i < size; i++ {
		path := filepath.Join(root, fmt.Sprintf("%05d.md", i))
		writeScanTestNote(b, path, fmt.Sprintf("note-%05d", i), fmt.Sprintf("Note %05d", i), fmt.Sprintf("note-%05d", i), now)
	}
}
