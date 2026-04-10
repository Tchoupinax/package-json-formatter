package config

import (
	"fmt"
	"maps"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// EnsureKeysResult is the merged ensure-keys map for one package.json (only missing keys are applied later in format).
type EnsureKeysResult struct {
	// Skip is true when ensureKeysIgnore matches: no keys are ensured.
	Skip bool
	// Keys to insert when absent (merged from ensureKeys and ensureKeysFiles).
	Keys map[string]interface{}
}

// EnsureKeysFor resolves ensureKeys, ensureKeysIgnore, and ensureKeysFiles for one package.json path.
func (c Config) EnsureKeysFor(packageJSONPath, anchorDir, workingDir string) (EnsureKeysResult, error) {
	rel, err := packageJSONRel(packageJSONPath, anchorDir, workingDir)
	if err != nil {
		return EnsureKeysResult{}, err
	}

	for _, pattern := range c.EnsureKeysIgnore {
		pattern = filepath.ToSlash(filepath.Clean(pattern))
		ok, err := doublestar.Match(pattern, rel)
		if err != nil {
			return EnsureKeysResult{}, fmt.Errorf("ensureKeysIgnore pattern %q: %w", pattern, err)
		}
		if ok {
			return EnsureKeysResult{Skip: true}, nil
		}
	}

	merged := make(map[string]interface{})
	if c.EnsureKeys != nil {
		maps.Copy(merged, c.EnsureKeys)
	}
	for i, rule := range c.EnsureKeysFiles {
		if rule.Path == "" {
			continue
		}
		pattern := filepath.ToSlash(filepath.Clean(rule.Path))
		ok, err := doublestar.Match(pattern, rel)
		if err != nil {
			return EnsureKeysResult{}, fmt.Errorf("ensureKeysFiles[%d] pattern %q: %w", i, rule.Path, err)
		}
		if ok && rule.Keys != nil {
			maps.Copy(merged, rule.Keys)
		}
	}

	return EnsureKeysResult{Skip: false, Keys: merged}, nil
}
