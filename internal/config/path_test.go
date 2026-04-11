package config

import (
	"path/filepath"
	"testing"
)

func TestPackageJSONRel_RelativePackagePath(t *testing.T) {
	wd := t.TempDir()
	pkgRel := filepath.Join("packages", "transformations", "package.json")
	got, err := packageJSONRel(pkgRel, wd, wd)
	if err != nil {
		t.Fatal(err)
	}
	want := "packages/transformations/package.json"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPackageJSONRel_RelativeAnchorDot(t *testing.T) {
	wd := t.TempDir()
	pkgRel := filepath.Join("packages", "a", "package.json")
	got, err := packageJSONRel(pkgRel, ".", wd)
	if err != nil {
		t.Fatal(err)
	}
	want := "packages/a/package.json"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestScriptMergeFor_RelativePackageMatchesPatterns(t *testing.T) {
	wd := t.TempDir()
	pkgRel := filepath.Join("packages", "app", "package.json")
	c := Config{
		Scripts: map[string]string{"build": "tsc"},
		ScriptsFiles: []ScriptsFileRule{
			{Path: "packages/app/package.json", Scripts: map[string]string{"test": "vitest"}},
		},
	}
	sm, err := c.ScriptMergeFor(pkgRel, wd, wd)
	if err != nil {
		t.Fatal(err)
	}
	if sm.Skip {
		t.Fatal("expected not skipped")
	}
	if sm.Overrides["build"] != "tsc" || sm.Overrides["test"] != "vitest" {
		t.Fatalf("overrides: %#v", sm.Overrides)
	}
}
