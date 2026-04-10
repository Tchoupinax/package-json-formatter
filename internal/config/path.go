package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// packageJSONRel returns the path of packageJSONPath relative to the config directory,
// using workingDir when anchorDir is empty. Paths use forward slashes.
func packageJSONRel(packageJSONPath, anchorDir, workingDir string) (string, error) {
	anchor := filepath.Clean(anchorDir)
	if anchor == "" {
		anchor = filepath.Clean(workingDir)
	}
	pkg := filepath.Clean(packageJSONPath)
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
