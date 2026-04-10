package config

import (
	"fmt"
	"maps"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// ScriptMerge describes how to apply config scripts to one package.json.
type ScriptMerge struct {
	// Skip is true when this file matches scriptsIgnore: scripts are left unchanged.
	Skip bool
	// Overrides are merged into existing scripts when Skip is false (later keys win).
	Overrides map[string]string
}

// ScriptMergeFor resolves global scripts, scriptsIgnore, and scriptsFiles for one package.json path.
// anchorDir is the config file directory; if empty, workingDir is used to resolve relative patterns.
func (c Config) ScriptMergeFor(packageJSONPath, anchorDir, workingDir string) (ScriptMerge, error) {
	rel, err := packageJSONRel(packageJSONPath, anchorDir, workingDir)
	if err != nil {
		return ScriptMerge{}, err
	}

	for _, pattern := range c.ScriptsIgnore {
		pattern = filepath.ToSlash(filepath.Clean(pattern))
		ok, err := doublestar.Match(pattern, rel)
		if err != nil {
			return ScriptMerge{}, fmt.Errorf("scriptsIgnore pattern %q: %w", pattern, err)
		}
		if ok {
			return ScriptMerge{Skip: true}, nil
		}
	}

	merged := make(map[string]string)
	if c.Scripts != nil {
		maps.Copy(merged, c.Scripts)
	}
	for i, rule := range c.ScriptsFiles {
		if rule.Path == "" {
			continue
		}
		pattern := filepath.ToSlash(filepath.Clean(rule.Path))
		ok, err := doublestar.Match(pattern, rel)
		if err != nil {
			return ScriptMerge{}, fmt.Errorf("scriptsFiles[%d] pattern %q: %w", i, rule.Path, err)
		}
		if ok && rule.Scripts != nil {
			maps.Copy(merged, rule.Scripts)
		}
	}

	return ScriptMerge{Skip: false, Overrides: merged}, nil
}
