package config

import (
	"path/filepath"
	"testing"
)

func TestEnsureKeysFor_Merge(t *testing.T) {
	cfgDir := t.TempDir()
	pkg := filepath.Join(cfgDir, "packages", "a", "package.json")
	c := Config{
		EnsureKeys: map[string]interface{}{"prettier": true},
		EnsureKeysFiles: []EnsureKeysFileRule{
			{Path: "packages/a/package.json", Keys: map[string]interface{}{"private": true}},
		},
	}
	r, err := c.EnsureKeysFor(pkg, cfgDir, cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Skip {
		t.Fatal("expected not skipped")
	}
	if r.Keys["prettier"] != true || r.Keys["private"] != true {
		t.Fatalf("keys: %#v", r.Keys)
	}
}

func TestEnsureKeysFor_Ignore(t *testing.T) {
	cfgDir := t.TempDir()
	pkg := filepath.Join(cfgDir, "vendor", "x", "package.json")
	c := Config{
		EnsureKeys:       map[string]interface{}{"x": 1},
		EnsureKeysIgnore: []string{"vendor/**"},
	}
	r, err := c.EnsureKeysFor(pkg, cfgDir, cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Skip {
		t.Fatal("expected skipped")
	}
}
