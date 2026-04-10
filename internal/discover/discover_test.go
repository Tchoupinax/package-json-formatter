package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageJSONFiles_SkipsNodeModules(t *testing.T) {
	root := t.TempDir()
	mustMk(t, filepath.Join(root, "packages", "a"), `{"name":"a"}`)
	mustMk(t, filepath.Join(root, "packages", "b"), `{"name":"b"}`)
	nm := filepath.Join(root, "packages", "a", "node_modules", "x")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, PackageJSON), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := PackageJSONFiles(root, []string{"node_modules"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("got %d paths: %v", len(paths), paths)
	}
}

func mustMk(t *testing.T, dir string, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PackageJSON), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
