package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var notesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a note",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, err := cmd.Flags().GetString("title")
		if err != nil {
			return err
		}
		projectSelector, err := cmd.Flags().GetString("project")
		if err != nil {
			return err
		}
		useStdin, err := cmd.Flags().GetBool("stdin")
		if err != nil {
			return err
		}
		bodyFile, err := cmd.Flags().GetString("body-file")
		if err != nil {
			return err
		}
		tags, err := cmd.Flags().GetStringArray("tag")
		if err != nil {
			return err
		}
		if title == "" {
			return app.NewCLIUsageError("note title is required", nil)
		}
		if useStdin && bodyFile != "" {
			return app.NewCLIUsageError("--stdin and --body-file cannot be combined", nil)
		}

		root, err := resolveNotesProjectRoot(projectSelector)
		if err != nil {
			return err
		}

		var body []byte
		switch {
		case bodyFile != "":
			body, err = os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
		case useStdin:
			body, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
		}

		created, err := notes.Create(notes.CreateInput{
			RootDir: root,
			Title:   title,
			Body:    body,
			Tags:    tags,
		})
		if err != nil {
			return err
		}

		output := notesCreateOutput{
			NoteID:      created.NoteID,
			Slug:        created.Slug,
			Path:        created.Path,
			ContentHash: created.ContentHash,
		}
		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s created\n", created.Path), output)
	},
}

func init() {
	notesCreateCmd.Flags().String("project", "", "select a project")
	mustRegisterProjectFlagCompletion(notesCreateCmd, "project")
	notesCreateCmd.Flags().String("title", "", "note title")
	notesCreateCmd.Flags().Bool("stdin", false, "read the note body from stdin")
	notesCreateCmd.Flags().String("body-file", "", "read the note body from a file")
	notesCreateCmd.Flags().StringArray("tag", nil, "add a note tag")
	notesCmd.AddCommand(notesCreateCmd)
}

type notesCreateOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

func resolveNotesProjectRoot(projectSelector string) (string, error) {
	container, err := mustAppContainer()
	if err != nil {
		return "", err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	resolvedProject, err := project.ResolveProject(project.ResolveProjectInput{
		CWD:             cwd,
		ProjectSelector: projectSelector,
	})
	if err != nil {
		return "", err
	}

	return project.ResolveMemoriesRoot(project.MemoriesRootInput{
		Kind:         string(resolvedProject.Project.Kind),
		Slug:         resolvedProject.Project.Slug,
		MemoriesPath: resolvedProject.Project.MemoriesPath,
		MemoriesHome: container.Paths.MemoriesHome,
		RepoRoot:     filepath.Dir(resolvedProject.MnemonicFilePath),
	})
}
