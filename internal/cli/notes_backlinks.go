package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/graph"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var notesBacklinksCmd = &cobra.Command{
	Use:   "backlinks SELECTOR",
	Short: "Show backlinks for a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectSelector, err := cmd.Flags().GetString("project")
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		resolvedProject, err := project.ResolveProject(project.ResolveProjectInput{
			CWD:             cwd,
			ProjectSelector: projectSelector,
		})
		if err != nil {
			return err
		}

		repoRoot := filepath.Dir(resolvedProject.MnemonicFilePath)
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		_, err = project.ResolveMemoriesRoot(project.MemoriesRootInput{
			Kind:         string(resolvedProject.Project.Kind),
			Slug:         resolvedProject.Project.Slug,
			MemoriesPath: resolvedProject.Project.MemoriesPath,
			MemoriesHome: container.Paths.MemoriesHome,
			RepoRoot:     repoRoot,
		})
		if err != nil {
			return err
		}

		indexPath, err := index.Path(resolvedProject.Project.ID)
		if err != nil {
			return err
		}
		if _, err := os.Stat(indexPath); err != nil {
			if os.IsNotExist(err) {
				return app.NewNotFoundError("index missing; run `mnemonic project reindex`", nil)
			}
			return fmt.Errorf("stat index %q: %w", indexPath, err)
		}

		db, err := sql.Open("sqlite", indexPath)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		target, err := queryIndexedNoteBySelector(db, args[0])
		if err != nil {
			return err
		}

		links, err := graph.Backlinks(db, target.NoteID)
		if err != nil {
			return err
		}

		output := notesBacklinksOutput{Links: links}
		if output.Links == nil {
			output.Links = []graph.Backlink{}
		}
		human := fmt.Sprintf("%d links\n", len(output.Links))
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	notesBacklinksCmd.Flags().String("project", "", "select a project")
	mustRegisterProjectFlagCompletion(notesBacklinksCmd, "project")
	notesCmd.AddCommand(notesBacklinksCmd)
}

type notesBacklinksOutput struct {
	Links []graph.Backlink `json:"links"`
}

type indexedNote struct {
	NoteID string
	Slug   string
	Title  string
	Path   string
}

func queryIndexedNoteBySelector(db *sql.DB, selector string) (indexedNote, error) {
	row := db.QueryRow(
		`SELECT note_id, slug, title, rel_path
		 FROM notes
		 WHERE note_id = ? OR slug = ? OR rel_path = ? OR title = ?`,
		selector, selector, selector, selector,
	)
	var note indexedNote
	if err := row.Scan(&note.NoteID, &note.Slug, &note.Title, &note.Path); err != nil {
		if err == sql.ErrNoRows {
			return indexedNote{}, app.NewNotFoundError(fmt.Sprintf("note %q not found", selector), nil)
		}
		return indexedNote{}, fmt.Errorf("query note %q: %w", selector, err)
	}
	return note, nil
}
