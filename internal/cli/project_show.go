package cli

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/spf13/cobra"
)

var projectShowCmd = &cobra.Command{
	Use:   "show NAME_OR_UUID",
	Short: "Show a registered project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		project, err := loadProjectBySelector(container.Paths, args[0])
		if err != nil {
			return err
		}

		human := fmt.Sprintf("%s %s\n", project.ProjectID, project.Name)
		return PrintOutput(cmd.OutOrStdout(), human, project)
	},
}

func init() {
	projectCmd.AddCommand(projectShowCmd)
}

type projectShowOutput struct {
	ProjectID string                `json:"project_id"`
	Name      string                `json:"name"`
	Slug      string                `json:"slug"`
	Kind      string                `json:"kind"`
	StatePath string                `json:"state_path"`
	Location  projectLocationOutput `json:"location"`
	Status    projectStatusOutput   `json:"status"`
}

type projectLocationOutput struct {
	MnemonicFileAbs string `json:"mnemonic_file_abs"`
	RepoRootAbs     string `json:"repo_root_abs"`
	MemoriesAbs     string `json:"memories_abs"`
	ManifestAbs     string `json:"manifest_abs"`
	SourceKind      string `json:"source_kind"`
	LastSeenAt      string `json:"last_seen_at"`
}

type projectStatusOutput struct {
	IndexSchemaVersion int    `json:"index_schema_version"`
	LastSeenAt         string `json:"last_seen_at"`
	IndexPresent       bool   `json:"index_present"`
	NeedsReindex       bool   `json:"needs_reindex"`
}

func loadProjectBySelector(effectivePaths paths.EffectivePaths, selector string) (projectShowOutput, error) {
	record, err := queryProjectBySelector(selector)
	if err != nil {
		return projectShowOutput{}, err
	}

	return projectShowOutput{
		ProjectID: record.ProjectID,
		Name:      record.Name,
		Slug:      record.Slug,
		Kind:      record.Kind,
		StatePath: filepath.Join(effectivePaths.StateHome, "mnemonic", "projects", record.ProjectID, "state.toml"),
		Location: projectLocationOutput{
			MnemonicFileAbs: record.Location.mnemonicFileAbs,
			RepoRootAbs:     record.Location.repoRootAbs,
			MemoriesAbs:     record.Location.memoriesAbs,
			ManifestAbs:     record.Location.manifestAbs,
			SourceKind:      record.Location.sourceKind,
			LastSeenAt:      record.Location.lastSeenAt,
		},
		Status: projectStatusOutput{
			IndexSchemaVersion: record.Status.indexSchemaVersion,
			LastSeenAt:         record.Status.lastSeenAt,
			IndexPresent:       record.Status.indexPresent,
			NeedsReindex:       record.Status.needsReindex,
		},
	}, nil
}

type queryProjectLocation struct {
	mnemonicFileAbs string
	repoRootAbs     string
	memoriesAbs     string
	manifestAbs     string
	sourceKind      string
	lastSeenAt      string
}

type queryProjectStatus struct {
	indexSchemaVersion int
	lastSeenAt         string
	indexPresent       bool
	needsReindex       bool
}

func queryProjectBySelector(selector string) (projectLookupResult, error) {
	container, err := mustAppContainer()
	if err != nil {
		return projectLookupResult{}, err
	}
	db := container.Services.Registry

	record := projectLookupResult{}
	err = db.QueryRow(
		`SELECT p.project_id, p.name, p.slug, p.kind,
		        COALESCE(l.mnemonic_file_abs, ''),
		        COALESCE(l.repo_root_abs, ''),
		        l.memories_abs,
		        COALESCE(l.manifest_abs, ''),
		        l.source_kind,
		        l.last_seen_at,
		        s.index_schema_version,
		        s.last_seen_at,
		        s.index_present,
		        s.needs_reindex
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 JOIN project_status s ON s.project_id = p.project_id
		 WHERE p.removed_at IS NULL AND p.project_id = ?`,
		selector,
	).Scan(&record.ProjectID, &record.Name, &record.Slug, &record.Kind, &record.Location.mnemonicFileAbs, &record.Location.repoRootAbs, &record.Location.memoriesAbs, &record.Location.manifestAbs, &record.Location.sourceKind, &record.Location.lastSeenAt, &record.Status.indexSchemaVersion, &record.Status.lastSeenAt, &record.Status.indexPresent, &record.Status.needsReindex)
	if err == sql.ErrNoRows {
		err = db.QueryRow(
			`SELECT p.project_id, p.name, p.slug, p.kind,
			        COALESCE(l.mnemonic_file_abs, ''),
			        COALESCE(l.repo_root_abs, ''),
			        l.memories_abs,
			        COALESCE(l.manifest_abs, ''),
			        l.source_kind,
			        l.last_seen_at,
			        s.index_schema_version,
			        s.last_seen_at,
			        s.index_present,
			        s.needs_reindex
			 FROM projects p
			 JOIN project_locations l ON l.project_id = p.project_id
			 JOIN project_status s ON s.project_id = p.project_id
			 WHERE p.removed_at IS NULL AND p.slug = ?`,
			selector,
		).Scan(&record.ProjectID, &record.Name, &record.Slug, &record.Kind, &record.Location.mnemonicFileAbs, &record.Location.repoRootAbs, &record.Location.memoriesAbs, &record.Location.manifestAbs, &record.Location.sourceKind, &record.Location.lastSeenAt, &record.Status.indexSchemaVersion, &record.Status.lastSeenAt, &record.Status.indexPresent, &record.Status.needsReindex)
	}
	if err == sql.ErrNoRows {
		return projectLookupResult{}, app.NewNotFoundError(fmt.Sprintf("project %q not found", selector), nil)
	}
	if err != nil {
		return projectLookupResult{}, fmt.Errorf("query project %q: %w", selector, err)
	}

	return record, nil
}

type projectLookupResult struct {
	ProjectID string
	Name      string
	Slug      string
	Kind      string
	Location  queryProjectLocation
	Status    queryProjectStatus
}
