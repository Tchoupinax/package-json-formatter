package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"package-json-formatter/internal/config"
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

	return runPaths(wd, paths, *write, cfg, cfgDir)
}

func runPaths(wd string, paths []string, write bool, cfg config.Config, cfgDir string) int {
	ui := pjfcli.New(wd)
	ui.Header(len(paths))
	runStart := time.Now()

	var failed bool
	var okN, failN int
	for _, p := range paths {
		if ok := processPath(p, write, cfg, cfgDir, wd, ui); !ok {
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

func processPath(path string, write bool, cfg config.Config, cfgDir, wd string, ui *pjfcli.UI) bool {
	start := time.Now()
	raw, err := os.ReadFile(path)
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("read: %v", err))
		return false
	}
	sm, err := cfg.ScriptMergeFor(path, cfgDir, wd)
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("scripts: %v", err))
		return false
	}
	ek, err := cfg.EnsureKeysFor(path, cfgDir, wd)
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("ensureKeys: %v", err))
		return false
	}
	out, err := format.Format(raw, cfg.KeyOrder, sm.Skip, sm.Overrides, ek.Skip, ek.Keys, cfg.PinDependencyVersionsEnabled())
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("format: %v", err))
		return false
	}
	if write {
		if err := os.WriteFile(path, out, 0o644); err != nil { //nolint:gosec // G306: package.json stays group/world-readable like typical npm projects
			ui.Row(false, path, time.Since(start), fmt.Sprintf("write: %v", err))
			return false
		}
		ui.Row(true, path, time.Since(start), "")
		return true
	}
	os.Stdout.Write(out)
	ui.Row(true, path, time.Since(start), "")
	return true
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
