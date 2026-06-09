package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		projectSelector := ""
		if len(args) == 1 {
			projectSelector = args[0]
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		resolved, err := project.ResolveProject(project.ResolveProjectInput{
			CWD:             cwd,
			ProjectSelector: projectSelector,
		})
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		repoRoot := filepath.Dir(resolved.MnemonicFilePath)
		memoriesRoot, err := project.ResolveMemoriesRoot(project.MemoriesRootInput{
			Kind:         string(resolved.Project.Kind),
			Slug:         resolved.Project.Slug,
			MemoriesPath: resolved.Project.MemoriesPath,
			MemoriesHome: container.Paths.MemoriesHome,
			RepoRoot:     repoRoot,
		})
		if err != nil {
			return err
		}

		result, err := doctorProject(resolved, repoRoot, memoriesRoot)
		if err != nil {
			return err
		}
		human := fmt.Sprintf("%s\n", result.Status)
		return PrintOutput(cmd.OutOrStdout(), human, result)
	},
}

func init() {
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

func doctorProject(resolved project.ResolvedProject, repoRoot, memoriesRoot string) (doctorResult, error) {
	result := doctorResult{Status: "ok"}
	result.addCheck(doctorCheck{Name: "registry schema", Status: doctorRegistrySchemaCheck()})
	result.addCheck(doctorCheck{Name: "project path exists", Status: pathStatus(repoRoot)})

	if resolved.Project.Kind == project.ProjectKindLocal || resolved.Project.Kind == project.ProjectKindRegular {
		result.addCheck(doctorParseCheck(".mnemonic", resolved.MnemonicFilePath))
	} else {
		result.addCheck(doctorParseCheck("mnemonic.toml", filepath.Join(memoriesRoot, "mnemonic.toml")))
	}

	indexPath, err := index.Path(resolved.Project.ID)
	if err != nil {
		return doctorResult{}, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			result.Status = "needs_reindex"
			result.addCheck(doctorCheck{Name: "index exists", Status: "missing"})
			return result, nil
		}
		return doctorResult{}, err
	}
	result.addCheck(doctorCheck{Name: "index exists", Status: "ok"})

	if err := index.QuickCheck(indexPath); err != nil {
		return doctorResult{}, err
	}
	result.addCheck(doctorCheck{Name: "index quick_check", Status: "ok"})

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
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

	dupUUIDs, dupSlugs, unresolved, trashIgnored, err := doctorNoteChecks(memoriesRoot, db)
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

func doctorRegistrySchemaCheck() string {
	path, err := registry.RegistryPath()
	if err != nil {
		return "error"
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		return "error"
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return "error"
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		return "error"
	}
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return "error"
	}
	if version != 1 {
		return "error"
	}
	for _, table := range []string{"registry_meta", "projects", "project_locations", "project_status"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name); err != nil {
			return "error"
		}
	}
	return "ok"
}

func doctorParseCheck(name, path string) doctorCheck {
	data, err := os.ReadFile(path)
	if err != nil {
		return doctorCheck{Name: name, Status: "missing", Detail: err.Error()}
	}
	if name == ".mnemonic" {
		if _, err := project.ParseMnemonicFile(data); err != nil {
			return doctorCheck{Name: name, Status: "error", Detail: err.Error()}
		}
		return doctorCheck{Name: name, Status: "ok"}
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
