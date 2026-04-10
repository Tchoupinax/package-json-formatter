package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"package-json-formatter/internal/config"
	"package-json-formatter/internal/discover"
	"package-json-formatter/internal/format"
	"package-json-formatter/internal/pjfcli"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		printVersion()
		return 0
	}

	configPath := flag.String("config", "", "YAML config path (default: pjf.yaml in working directory if that file exists)")
	write := flag.Bool("w", false, "write formatted output back to files")
	recursive := flag.Bool("r", true, "find all package.json under each target (monorepo mode)")
	showVersion := flag.Bool("version", false, "print version and exit")
	showVerShort := flag.Bool("v", false, "print version and exit")
	flag.Parse()
	if *showVersion || *showVerShort {
		printVersion()
		return 0
	}
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		return 1
	}

	effectiveConfig, err := resolveConfigPath(*configPath, wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}

	cfg := config.Default()
	cfgDir := ""
	if effectiveConfig != "" {
		loaded, err := config.Load(effectiveConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			return 1
		}
		cfg = loaded
		cfgDir = filepath.Dir(filepath.Clean(effectiveConfig))
	}

	paths, err := resolveTargets(args, cfg, cfgDir, *recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	if !*write && len(paths) > 1 {
		fmt.Fprintln(os.Stderr, "multiple package.json files: use -w to write, or pass a single directory/file")
		return 1
	}

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "no package.json files found")
		return 1
	}

	ui := pjfcli.New(wd)
	ui.Header(len(paths))
	runStart := time.Now()

	var failed bool
	var okN, failN int
	for _, p := range paths {
		bad := func() bool {
			start := time.Now()
			var detail string

			raw, err := os.ReadFile(p)
			if err != nil {
				detail = fmt.Sprintf("read: %v", err)
				ui.Row(false, p, time.Since(start), detail)
				return true
			}
			sm, err := cfg.ScriptMergeFor(p, cfgDir, wd)
			if err != nil {
				detail = fmt.Sprintf("scripts: %v", err)
				ui.Row(false, p, time.Since(start), detail)
				return true
			}
			ek, err := cfg.EnsureKeysFor(p, cfgDir, wd)
			if err != nil {
				detail = fmt.Sprintf("ensureKeys: %v", err)
				ui.Row(false, p, time.Since(start), detail)
				return true
			}
			out, err := format.Format(raw, cfg.KeyOrder, sm.Skip, sm.Overrides, ek.Skip, ek.Keys, cfg.PinDependencyVersionsEnabled())
			if err != nil {
				detail = fmt.Sprintf("format: %v", err)
				ui.Row(false, p, time.Since(start), detail)
				return true
			}
			if *write {
				if err := os.WriteFile(p, out, 0o644); err != nil {
					detail = fmt.Sprintf("write: %v", err)
					ui.Row(false, p, time.Since(start), detail)
					return true
				}
				ui.Row(true, p, time.Since(start), "")
				return false
			}
			os.Stdout.Write(out)
			ui.Row(true, p, time.Since(start), "")
			return false
		}()
		if bad {
			failed = true
			failN++
		} else {
			okN++
		}
	}
	ui.Footer(len(paths), okN, failN, time.Since(runStart))
	if failed {
		return 1
	}
	return 0
}

const defaultConfigFile = "pjf.yml"

// resolveConfigPath returns the config file to load, or "" if none.
// If -config is set, that path must exist. If unset, pjf.yaml in wd is used when present.
func resolveConfigPath(flagValue, wd string) (string, error) {
	if flagValue != "" {
		flagValue = filepath.Clean(flagValue)
		st, err := os.Stat(flagValue)
		if err != nil {
			return "", err
		}
		if st.IsDir() {
			return "", fmt.Errorf("%s: config path is a directory", flagValue)
		}
		return flagValue, nil
	}
	candidate := filepath.Join(wd, defaultConfigFile)
	st, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("%s: config path is a directory", candidate)
	}
	return candidate, nil
}

func resolveTargets(args []string, cfg config.Config, cfgDir string, recursive bool) ([]string, error) {
	if len(cfg.Roots) > 0 && cfgDir == "" {
		return nil, fmt.Errorf("config sets \"roots\" but no config file was loaded (roots are relative to the config file); add %s or pass -config", defaultConfigFile)
	}

	if len(cfg.Roots) > 0 {
		var absRoots []string
		for _, r := range cfg.Roots {
			r = filepath.Clean(r)
			if !filepath.IsAbs(r) {
				if cfgDir == "" {
					return nil, fmt.Errorf("internal: cfgDir empty with roots")
				}
				r = filepath.Join(cfgDir, r)
			}
			absRoots = append(absRoots, r)
		}
		return discover.FromRoots(absRoots, cfg.SkipDirNames)
	}

	if !recursive {
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
