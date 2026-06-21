package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/spf13/cobra"
)

var projectDoctorCmd = &cobra.Command{
	Use:               "doctor [NAME_OR_UUID]",
	Short:             "Run project health checks",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		projectSelector := projectSelectorValue()
		if len(args) == 1 {
			projectSelector = args[0]
		}
		if all && len(args) > 0 {
			return fmt.Errorf("--all cannot be combined with a project selector")
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		if all {
			if container.Services.Maint == nil {
				return fmt.Errorf("maintenance service is not configured")
			}
			result, err := container.Services.Maint.DoctorAll(cmd.Context())
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%d projects checked\n", len(result.Projects))
			return PrintOutput(cmd.OutOrStdout(), human, result)
		}

		runtime, err := runtimeAppForSelector(cmd.Context(), projectSelector)
		if err != nil {
			return err
		}

		repoRoot := runtime.KB.RepoRootDir
		if repoRoot == "" && runtime.KB.ManifestPath != "" {
			repoRoot = filepath.Dir(runtime.KB.ManifestPath)
		}

		result, err := doctorProject(container.Paths.MemoriesHome, runtime.KB, repoRoot)
		if err != nil {
			return err
		}
		human := fmt.Sprintf("%s\n", result.Status)
		return PrintOutput(cmd.OutOrStdout(), human, result)
	},
}

func init() {
	projectDoctorCmd.Flags().Bool("all", false, "run doctor across all active projects")
	projectCmd.AddCommand(projectDoctorCmd)
}

type doctorResult struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Count  int    `json:"count,omitempty"`
}

func doctorProject(memoriesHome string, resolved kb.KnowledgeBase, repoRoot string) (doctorResult, error) {
	result := doctorResult{Status: "ok"}
	result.addCheck(doctorCheck{Name: "registry file-based", Status: doctorRegistryFileCheck(memoriesHome)})
	result.addCheck(doctorCheck{Name: "project path exists", Status: pathStatus(repoRoot)})

	if resolved.Kind != string(project.ProjectKindLocal) {
		result.addCheck(doctorParseCheck("mnemonic.toml", resolved.ManifestPath))
	}

	if _, err := os.Stat(resolved.IndexPath); err != nil {
		if os.IsNotExist(err) {
			result.Status = "needs_reindex"
			result.addCheck(doctorCheck{Name: "index exists", Status: "missing"})
			return result, nil
		}
		return doctorResult{}, err
	}
	result.addCheck(doctorCheck{Name: "index exists", Status: "ok"})

	if err := index.QuickCheck(resolved.IndexPath); err != nil {
		return doctorResult{}, err
	}
	result.addCheck(doctorCheck{Name: "index quick_check", Status: "ok"})

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(resolved.IndexPath)+"?mode=ro")
	if err != nil {
		return doctorResult{}, err
	}
	defer func() { _ = db.Close() }()
	schemaStatus, err := index.CheckSchemaStatus(db)
	if err != nil {
		return doctorResult{}, err
	}
	if schemaStatus != index.SchemaStatusOK {
		result.Status = "needs_reindex"
		result.addCheck(doctorCheck{Name: "index schema", Status: "needs_reindex"})
		return result, nil
	}
	result.addCheck(doctorCheck{Name: "index schema", Status: "ok"})

	dupUUIDs, dupSlugs, unresolved, trashIgnored, err := doctorNoteChecks(resolved.RootDir, db)
	if err != nil {
		return doctorResult{}, err
	}
	result.addCheck(doctorCheck{Name: "duplicate note UUIDs", Status: countStatus(dupUUIDs), Count: dupUUIDs})
	result.addCheck(doctorCheck{Name: "duplicate note slugs", Status: countStatus(dupSlugs), Count: dupSlugs})
	result.addCheck(doctorCheck{Name: "unresolved link count", Status: countStatus(unresolved), Count: unresolved})
	result.addCheck(doctorCheck{Name: ".trash ignored", Status: countStatus(trashIgnored), Count: trashIgnored})
	result.addCheck(doctorStaleTempCheck(repoRoot))

	return result, nil
}

func (r *doctorResult) addCheck(check doctorCheck) {
	r.Checks = append(r.Checks, check)
	if r.Status == "ok" && check.Status == "warning" {
		r.Status = "warning"
	}
}

func countStatus(count int) string {
	if count == 0 {
		return "ok"
	}
	return "warning"
}

func pathStatus(path string) string {
	if _, err := os.Stat(path); err == nil {
		return "ok"
	}
	return "missing"
}

func doctorRegistryFileCheck(memoriesHome string) string {
	entries, issues, err := registry.Scan(memoriesHome)
	if err != nil {
		return "error"
	}
	for _, issue := range issues {
		if issue.Corrupt || issue.Orphan {
			return "warning"
		}
	}
	if len(entries) == 0 {
		return "ok"
	}
	return "ok"
}

func doctorParseCheck(name, path string) doctorCheck {
	if path == "" {
		return doctorCheck{Name: name, Status: "missing", Detail: "no manifest path recorded"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return doctorCheck{Name: name, Status: "missing", Detail: err.Error()}
	}
	if _, err := project.ParseMnemonicManifest(data); err != nil {
		return doctorCheck{Name: name, Status: "error", Detail: err.Error()}
	}
	return doctorCheck{Name: name, Status: "ok"}
}

func doctorNoteChecks(root string, db *sql.DB) (dupUUIDs int, dupSlugs int, unresolved int, trashIgnored int, err error) {
	paths, err := notes.Walk(root)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	seenUUID := map[string]int{}
	seenSlug := map[string]int{}
	for _, rel := range paths {
		if strings.Contains(rel, ".trash/") || strings.HasPrefix(rel, ".trash/") || rel == ".trash" {
			trashIgnored++
		}
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			return 0, 0, 0, 0, readErr
		}
		note, parseErr := markdown.ParseNote(data)
		if parseErr != nil {
			return 0, 0, 0, 0, parseErr
		}
		if note.MnemonicNoteID != "" {
			seenUUID[note.MnemonicNoteID]++
		}
		if slug := note.EffectiveSlug(); slug != "" {
			seenSlug[slug]++
		}
	}
	for _, count := range seenUUID {
		if count > 1 {
			dupUUIDs++
		}
	}
	for _, count := range seenSlug {
		if count > 1 {
			dupSlugs++
		}
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM links WHERE to_note_id IS NULL`).Scan(&unresolved); err != nil {
		return 0, 0, 0, 0, err
	}
	return dupUUIDs, dupSlugs, unresolved, trashIgnored, nil
}

func doctorStaleTempCheck(root string) doctorCheck {
	entries, err := os.ReadDir(root)
	if err != nil {
		return doctorCheck{Name: "stale temp files", Status: "error", Detail: err.Error()}
	}
	count := 0
	firstPath := ""
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".tmp") {
			continue
		}
		count++
		if firstPath == "" {
			firstPath = filepath.Join(root, name)
		}
	}
	check := doctorCheck{Name: "stale temp files", Status: countStatus(count), Count: count}
	if firstPath != "" {
		check.Detail = firstPath
	}
	return check
}
