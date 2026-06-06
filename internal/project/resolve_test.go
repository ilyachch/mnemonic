package project

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
)

func TestResolveProjectExplicitSelectorWinsOverEnv(t *testing.T) {
	cwd := t.TempDir()
	writeResolvableMnemonicFile(t, filepath.Join(cwd, ".mnemonic"))

	got, err := ResolveProject(ResolveProjectInput{
		CWD:              cwd,
		ProjectSelector:  "backend",
		EnvironmentValue: "infra",
	})
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if got.Project.Slug != "backend" {
		t.Fatalf("ResolveProject() slug = %q, want backend", got.Project.Slug)
	}
}

func TestResolveProjectEnvWinsOverNearest(t *testing.T) {
	cwd := t.TempDir()
	writeResolvableMnemonicFile(t, filepath.Join(cwd, ".mnemonic"))

	got, err := ResolveProject(ResolveProjectInput{
		CWD:              cwd,
		EnvironmentValue: "infra",
	})
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if got.Project.Slug != "infra" {
		t.Fatalf("ResolveProject() slug = %q, want infra", got.Project.Slug)
	}
}

func TestResolveProjectSingleProjectWithoutSelector(t *testing.T) {
	cwd := t.TempDir()
	writeTestMnemonicFile(t, filepath.Join(cwd, ".mnemonic"), "backend")

	got, err := ResolveProject(ResolveProjectInput{CWD: cwd})
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if got.Project.Slug != "backend" {
		t.Fatalf("ResolveProject() slug = %q, want backend", got.Project.Slug)
	}
}

func TestResolveProjectAmbiguousWithoutSelector(t *testing.T) {
	cwd := t.TempDir()
	writeResolvableMnemonicFile(t, filepath.Join(cwd, ".mnemonic"))

	_, err := ResolveProject(ResolveProjectInput{CWD: cwd})
	if err == nil {
		t.Fatal("ResolveProject() error = nil, want ambiguous")
	}
	if code := appErrorCode(err); code != app.CodeAmbiguous {
		t.Fatalf("ResolveProject() code = %v, want %v", code, app.CodeAmbiguous)
	}
}

func TestResolveProjectDetachedManifestDoesNotCountAsFallback(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	_ = memoriesHome
	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")
	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "backend"
	manifest.Slug = "backend"
	manifest.Kind = ManifestKindDetached
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.Generator.App = "mnemonic"
	if err := WriteMnemonicManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteMnemonicManifest() error = %v", err)
	}

	_, err := ResolveProject(ResolveProjectInput{CWD: cwd})
	if err == nil {
		t.Fatal("ResolveProject() error = nil, want not found")
	}
	if code := appErrorCode(err); code != app.CodeNotFound {
		t.Fatalf("ResolveProject() code = %v, want %v", code, app.CodeNotFound)
	}
}

func TestResolveProjectMissingEverythingReturnsNotFound(t *testing.T) {
	cwd := t.TempDir()

	_, err := ResolveProject(ResolveProjectInput{CWD: cwd})
	if err == nil {
		t.Fatal("ResolveProject() error = nil, want not found")
	}
	if code := appErrorCode(err); code != app.CodeNotFound {
		t.Fatalf("ResolveProject() code = %v, want %v", code, app.CodeNotFound)
	}
}

func writeResolvableMnemonicFile(t *testing.T, path string) {
	t.Helper()

	file := &MnemonicFile{
		Version:   1,
		CreatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		Projects: []MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  "backend",
				Slug:                  "backend",
				Kind:                  ProjectKindLocal,
				MemoriesPath:          filepath.Join(".mnemonic-memories", "backend"),
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440001",
				Name:                  "infra",
				Slug:                  "infra",
				Kind:                  ProjectKindRegular,
				MemoriesPath:          "infra",
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	if err := WriteMnemonicFile(path, file); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}
}

func appErrorCode(err error) app.ErrCode {
	var appErr *app.AppError
	if !asAppError(err, &appErr) {
		return app.CodeInternal
	}
	return appErr.Code
}

func asAppError(err error, target **app.AppError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*app.AppError); ok {
		*target = e
		return true
	}
	return false
}
