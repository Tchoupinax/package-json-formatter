package config

import (
	"path/filepath"
	"testing"
)

func TestScriptMergeFor_GlobalAndPerFile(t *testing.T) {
	cfgDir := t.TempDir()
	pkg := filepath.Join(cfgDir, "packages", "app", "package.json")
	c := Config{
		Scripts: map[string]string{"build": "tsc"},
		ScriptsFiles: []ScriptsFileRule{
			{Path: "packages/app/package.json", Scripts: map[string]string{"test": "vitest"}},
			{Path: "packages/**/package.json", Scripts: map[string]string{"lint": "eslint ."}},
		},
	}
	sm, err := c.ScriptMergeFor(pkg, cfgDir, cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if sm.Skip {
		t.Fatal("expected not skipped")
	}
	if sm.Overrides["build"] != "tsc" || sm.Overrides["test"] != "vitest" || sm.Overrides["lint"] != "eslint ." {
		t.Fatalf("overrides: %#v", sm.Overrides)
	}
}

func TestScriptMergeFor_Ignore(t *testing.T) {
	cfgDir := t.TempDir()
	pkg := filepath.Join(cfgDir, "vendor", "x", "package.json")
	c := Config{
		Scripts:       map[string]string{"build": "tsc"},
		ScriptsIgnore: []string{"vendor/**"},
	}
	sm, err := c.ScriptMergeFor(pkg, cfgDir, cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if !sm.Skip {
		t.Fatal("expected skipped")
	}
}

func TestScriptMergeFor_OutsideAnchorErrors(t *testing.T) {
	cfgDir := t.TempDir()
	pkg := filepath.Join(t.TempDir(), "other", "package.json")
	c := Config{Scripts: map[string]string{"x": "y"}}
	_, err := c.ScriptMergeFor(pkg, cfgDir, cfgDir)
	if err == nil {
		t.Fatal("expected error when package is not under anchor")
	}
}
