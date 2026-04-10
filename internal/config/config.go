package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config drives formatting and monorepo discovery.
type Config struct {
	// KeyOrder is the preferred top-level key order. Omitted keys follow in alphabetical order.
	KeyOrder []string `yaml:"keyOrder"`

	// Scripts merged into each package.json "scripts" object (overwrites same keys),
	// unless the file matches scriptsIgnore. Per-file rules in scriptsFiles apply after this.
	Scripts map[string]string `yaml:"scripts"`

	// ScriptsIgnore lists glob patterns (relative to the config file directory, forward slashes)
	// matching package.json paths for which no script merging is applied (global or scriptsFiles).
	ScriptsIgnore []string `yaml:"scriptsIgnore"`

	// ScriptsFiles applies extra script entries for matching package.json paths, in order;
	// later rules override earlier ones for the same script name.
	ScriptsFiles []ScriptsFileRule `yaml:"scriptsFiles"`

	// EnsureKeys lists top-level keys added only when missing (see ensureKeysIgnore, ensureKeysFiles).
	EnsureKeys map[string]interface{} `yaml:"ensureKeys"`

	// EnsureKeysIgnore skips ensureKeys for matching package.json paths (glob, relative to config dir).
	EnsureKeysIgnore []string `yaml:"ensureKeysIgnore"`

	// EnsureKeysFiles merges keys into ensureKeys for matching paths; later rules override earlier keys.
	EnsureKeysFiles []EnsureKeysFileRule `yaml:"ensureKeysFiles"`

	// Roots limits discovery to these directories (relative to the config file directory).
	// If empty, the CLI walks from each target path and finds every package.json (minus skips).
	Roots []string `yaml:"roots"`

	// SkipDirNames are directory names to ignore while walking (default includes node_modules, .git).
	SkipDirNames []string `yaml:"skipDirNames"`

	// PinDependencyVersions strips leading ^ and ~ in dependency maps. Omitted means true; set false to keep ranges.
	PinDependencyVersions *bool `yaml:"pinDependencyVersions"`
}

// ScriptsFileRule maps a glob pattern to scripts merged when the pattern matches.
type ScriptsFileRule struct {
	Path    string            `yaml:"path"`
	Scripts map[string]string `yaml:"scripts"`
}

// EnsureKeysFileRule adds top-level keys when missing for matching package.json paths.
type EnsureKeysFileRule struct {
	Path string                 `yaml:"path"`
	Keys map[string]interface{} `yaml:"keys"`
}

func Default() Config {
	return Config{
		KeyOrder: nil,
		Scripts:  nil,
		Roots:    nil,
		SkipDirNames: []string{
			"node_modules",
			".git",
			"dist",
			"build",
			"coverage",
			".turbo",
			".next",
			".nuxt",
		},
	}
}

func Load(path string) (Config, error) {
	c := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if len(c.SkipDirNames) == 0 {
		c.SkipDirNames = Default().SkipDirNames
	}
	return c, nil
}

// PinDependencyVersionsEnabled reports whether ^ and ~ should be stripped from dependency versions (default true).
func (c Config) PinDependencyVersionsEnabled() bool {
	if c.PinDependencyVersions == nil {
		return true
	}
	return *c.PinDependencyVersions
}
