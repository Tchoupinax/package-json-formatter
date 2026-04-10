package discover

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

const PackageJSON = "package.json"

// PackageJSONFiles returns every package.json path under root, excluding skip directories.
func PackageJSONFiles(root string, skipDirNames []string) ([]string, error) {
	skip := make(map[string]bool)
	for _, s := range skipDirNames {
		skip[strings.ToLower(s)] = true
	}

	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if path != root && skip[strings.ToLower(base)] {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == PackageJSON {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(out)
	return out, nil
}

// FromRoots runs PackageJSONFiles for each root and deduplicates.
func FromRoots(roots []string, skipDirNames []string) ([]string, error) {
	seen := make(map[string]bool)
	var all []string
	for _, r := range roots {
		r = filepath.Clean(r)
		list, err := PackageJSONFiles(r, skipDirNames)
		if err != nil {
			return nil, err
		}
		for _, p := range list {
			if !seen[p] {
				seen[p] = true
				all = append(all, p)
			}
		}
	}
	slices.Sort(all)
	return all, nil
}
