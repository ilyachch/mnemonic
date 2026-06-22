package cli

import "github.com/spf13/cobra"

func buildCommandTree() []*cobra.Command {
	config := cloneCommand(configCmd)
	configShow := cloneCommand(configShowCmd)
	config.AddCommand(configShow)

	hello := cloneCommand(helloCmd)

	mcp := cloneCommand(mcpCmd)
	mcp.Flags().Bool("read-only", false, "run the MCP server without write tools")

	notes := cloneCommand(notesCmd)
	notesBacklinks := cloneCommand(notesBacklinksCmd)
	notesCreate := cloneCommand(notesCreateCmd)
	notesCreate.Flags().String("title", "", "note title")
	notesCreate.Flags().Bool("stdin", false, "read the note body from stdin")
	notesCreate.Flags().String("body-file", "", "read the note body from a file")
	notesCreate.Flags().StringArray("tag", nil, "add a note tag")
	notesDelete := cloneCommand(notesDeleteCmd)
	notesDelete.Flags().Bool("dry-run", false, "show what would be deleted without making changes")
	notesDelete.Flags().Bool("hard", false, "delete the note file instead of moving it to trash")
	notesDelete.Flags().Bool("yes", false, "confirm deletion without prompting")
	notesEdit := cloneCommand(notesEditCmd)
	notesEdit.Flags().String("append", "", "append text to the note body")
	notesEdit.Flags().String("body-file", "", "replace the note body with the contents of a file")
	notesEdit.Flags().String("if-match", "", "only update if the current content hash matches")
	notesEdit.Flags().StringArray("set", nil, "set a frontmatter field")
	notesList := cloneCommand(notesListCmd)
	notesSearch := cloneCommand(notesSearchCmd)
	notesSearch.Flags().Int("limit", 20, "maximum number of results")
	notesSearch.Flags().String("tag", "", "filter by tag")
	notesShow := cloneCommand(notesShowCmd)
	notes.AddCommand(notesBacklinks)
	notes.AddCommand(notesCreate)
	notes.AddCommand(notesDelete)
	notes.AddCommand(notesEdit)
	notes.AddCommand(notesList)
	notes.AddCommand(notesSearch)
	notes.AddCommand(notesShow)

	project := cloneCommand(projectCmd)
	projectDoctor := cloneCommand(projectDoctorCmd)
	projectDoctor.Flags().Bool("all", false, "run doctor across all active projects")
	projectImport := cloneCommand(projectImportCmd)
	projectImport.Flags().Bool("dry-run", false, "Show what would be imported without making changes")
	projectInit := cloneCommand(projectInitCmd)
	projectInit.Flags().Bool("local", false, "create a local project")
	projectInit.Flags().String("description", "", "optional description of this memory's knowledge scope")
	projectList := cloneCommand(projectListCmd)
	projectReindex := cloneCommand(projectReindexCmd)
	projectReindex.Flags().Bool("all", false, "reindex all active projects")
	projectRemove := cloneCommand(projectRemoveCmd)
	projectRemove.Flags().Bool("wipe", false, "also remove all markdown notes")
	projectShow := cloneCommand(projectShowCmd)
	project.AddCommand(projectDoctor)
	project.AddCommand(projectImport)
	project.AddCommand(projectInit)
	project.AddCommand(projectList)
	project.AddCommand(projectReindex)
	project.AddCommand(projectRemove)
	project.AddCommand(projectShow)

	tags := cloneCommand(tagsCmd)
	tagsList := cloneCommand(tagsListCmd)
	tags.AddCommand(tagsList)

	version := cloneCommand(versionCmd)

	web := cloneCommand(webCmd)
	webServe := cloneCommand(webServeCmd)
	webServe.Flags().String("port", "", "listen port for the web server")
	webServe.Flags().String("addr", "", "listen address for the web server")
	_ = webServe.Flags().MarkHidden("addr")
	web.AddCommand(webServe)

	return []*cobra.Command{config, hello, mcp, notes, project, tags, version, web}
}

func cloneCommand(src *cobra.Command) *cobra.Command {
	if src == nil {
		return nil
	}
	return &cobra.Command{
		Use:                        src.Use,
		Aliases:                    append([]string(nil), src.Aliases...),
		SuggestFor:                 append([]string(nil), src.SuggestFor...),
		Short:                      src.Short,
		GroupID:                    src.GroupID,
		Long:                       src.Long,
		Example:                    src.Example,
		ValidArgs:                  src.ValidArgs,
		ValidArgsFunction:          src.ValidArgsFunction,
		Args:                       src.Args,
		ArgAliases:                 append([]string(nil), src.ArgAliases...),
		BashCompletionFunction:     src.BashCompletionFunction,
		Deprecated:                 src.Deprecated,
		Annotations:                cloneStringMap(src.Annotations),
		Version:                    src.Version,
		PersistentPreRun:           src.PersistentPreRun,
		PersistentPreRunE:          src.PersistentPreRunE,
		PreRun:                     src.PreRun,
		PreRunE:                    src.PreRunE,
		Run:                        src.Run,
		RunE:                       src.RunE,
		PostRun:                    src.PostRun,
		PostRunE:                   src.PostRunE,
		PersistentPostRun:          src.PersistentPostRun,
		PersistentPostRunE:         src.PersistentPostRunE,
		FParseErrWhitelist:         src.FParseErrWhitelist,
		CompletionOptions:          src.CompletionOptions,
		TraverseChildren:           src.TraverseChildren,
		Hidden:                     src.Hidden,
		SilenceErrors:              src.SilenceErrors,
		SilenceUsage:               src.SilenceUsage,
		DisableFlagParsing:         src.DisableFlagParsing,
		DisableAutoGenTag:          src.DisableAutoGenTag,
		DisableFlagsInUseLine:      src.DisableFlagsInUseLine,
		DisableSuggestions:         src.DisableSuggestions,
		SuggestionsMinimumDistance: src.SuggestionsMinimumDistance,
	}
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
