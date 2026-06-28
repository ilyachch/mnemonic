package cli

import "github.com/spf13/cobra"

func buildCommandTree() []*cobra.Command {
	config := newConfigCommand()
	config.AddCommand(newConfigShowCommand())

	stdio := newStdioCommand()
	mcp := newMCPCommand()

	notes := newNotesCommand()
	notes.AddCommand(newNotesBacklinksCommand())
	notes.AddCommand(newNotesCreateCommand())
	notes.AddCommand(newNotesDeleteCommand())
	notes.AddCommand(newNotesEditCommand())
	notes.AddCommand(newNotesListCommand())
	notes.AddCommand(newNotesSearchCommand())
	notes.AddCommand(newNotesShowCommand())

	project := newProjectCommand()
	project.AddCommand(newProjectDoctorCommand())
	project.AddCommand(newProjectImportCommand())
	project.AddCommand(newProjectInitCommand())
	project.AddCommand(newProjectListCommand())
	project.AddCommand(newProjectReindexCommand())
	project.AddCommand(newProjectRemoveCommand())
	project.AddCommand(newProjectShowCommand())

	tags := newTagsCommand()
	tags.AddCommand(newTagsListCommand())

	version := newVersionCommand()

	web := newWebCommand()
	web.AddCommand(newWebServeCommand())

	return []*cobra.Command{config, stdio, mcp, notes, project, tags, version, web}
}
