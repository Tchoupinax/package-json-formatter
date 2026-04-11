package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// absFromWD returns an absolute path: absolute inputs are cleaned; relative inputs are
// joined with workingDir (as filepath.WalkDir often yields paths relative to the walk root).
func absFromWD(p, workingDir string) string {
	p = filepath.Clean(p)
	wd := filepath.Clean(workingDir)
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(wd, p))
}

// packageJSONRel returns the path of packageJSONPath relative to the config directory,
// using workingDir when anchorDir is empty. Paths use forward slashes.
func packageJSONRel(packageJSONPath, anchorDir, workingDir string) (string, error) {
	wd := filepath.Clean(workingDir)
	anchor := filepath.Clean(anchorDir)
	if anchor == "" {
		anchor = wd
	} else {
		anchor = absFromWD(anchor, wd)
	}
	pkg := absFromWD(packageJSONPath, wd)
	rel, err := filepath.Rel(anchor, pkg)
	if err != nil {
		return "", fmt.Errorf("%s: not under config anchor %s: %w", pkg, anchor, err)
	}
	if relEscapesAnchor(rel) {
		return "", fmt.Errorf("%s: package.json must be under config directory %s", pkg, anchor)
	}
	return filepath.ToSlash(rel), nil
}

func relEscapesAnchor(rel string) bool {
	rel = filepath.ToSlash(rel)
	return rel == ".." || strings.HasPrefix(rel, "../")
}
