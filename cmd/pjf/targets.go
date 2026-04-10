package main

import (
	"fmt"
	"os"
	"path/filepath"

	"package-json-formatter/internal/config"
	"package-json-formatter/internal/discover"
)

func resolveTargets(args []string, cfg config.Config, cfgDir string, recursive bool) ([]string, error) {
	if len(cfg.Roots) > 0 {
		return targetsFromConfigRoots(cfg, cfgDir)
	}
	if !recursive {
		return targetsNonRecursive(args)
	}
	return targetsRecursive(args, cfg)
}

func targetsFromConfigRoots(cfg config.Config, cfgDir string) ([]string, error) {
	if cfgDir == "" {
		return nil, fmt.Errorf("config sets \"roots\" but no config file was loaded (roots are relative to the config file); add %s or pass -config", defaultConfigFile)
	}
	var absRoots []string
	for _, r := range cfg.Roots {
		r = filepath.Clean(r)
		if !filepath.IsAbs(r) {
			r = filepath.Join(cfgDir, r)
		}
		absRoots = append(absRoots, r)
	}
	return discover.FromRoots(absRoots, cfg.SkipDirNames)
}

func targetsNonRecursive(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		a = filepath.Clean(a)
		st, err := os.Stat(a)
		if err != nil {
			return nil, err
		}
		if st.IsDir() {
			pj := filepath.Join(a, discover.PackageJSON)
			if _, err := os.Stat(pj); err != nil {
				return nil, fmt.Errorf("%s: no package.json in directory", a)
			}
			out = append(out, pj)
			continue
		}
		if filepath.Base(a) != discover.PackageJSON {
			return nil, fmt.Errorf("%s: expected a package.json path or directory", a)
		}
		out = append(out, a)
	}
	return out, nil
}

func targetsRecursive(args []string, cfg config.Config) ([]string, error) {
	var all []string
	seen := make(map[string]bool)
	for _, a := range args {
		a = filepath.Clean(a)
		st, err := os.Stat(a)
		if err != nil {
			return nil, err
		}
		var roots []string
		if st.IsDir() {
			roots = []string{a}
		} else {
			if filepath.Base(a) != discover.PackageJSON {
				return nil, fmt.Errorf("%s: use a directory or package.json when -r is set", a)
			}
			roots = []string{filepath.Dir(a)}
		}
		list, err := discover.FromRoots(roots, cfg.SkipDirNames)
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
	return all, nil
}
