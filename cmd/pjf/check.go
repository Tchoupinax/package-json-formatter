package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"package-json-formatter/internal/check"
	"package-json-formatter/internal/config"
	"package-json-formatter/internal/pjfcli"
)

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "YAML config path (default: pjf.yml in working directory if that file exists)")
	recursive := fs.Bool("r", true, "find all package.json under each target (monorepo mode)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: pjf check [flags] [path ...]\n\n")
		fmt.Fprintf(fs.Output(), "Fail when package.json differs from what pjf would write:\n")
		fmt.Fprintf(fs.Output(), "  • key order / formatting (keyOrder, sorting, trailing newline) vs pjf output\n")
		fmt.Fprintf(fs.Output(), "  • script commands that conflict with merged scripts / scriptsFiles in pjf.yml\n")
		fmt.Fprintf(fs.Output(), "  • dependency versions that would be pinned (^ / ~ removed) unless pinDependencyVersions is false\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	flagArgs := fs.Args()
	if len(flagArgs) == 0 {
		flagArgs = []string{"."}
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

	paths, err := resolveTargets(flagArgs, cfg, cfgDir, *recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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
		if ok := checkPath(p, cfg, cfgDir, wd, ui); !ok {
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

func checkPath(path string, cfg config.Config, cfgDir, wd string, ui *pjfcli.UI) bool {
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
	scriptDrifts, err := check.ScriptOverrideDrifts(raw, sm.Skip, sm.Overrides)
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("parse: %v", err))
		return false
	}
	pinDrifts, err := check.PinDrifts(raw, cfg.PinDependencyVersionsEnabled())
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("parse: %v", err))
		return false
	}
	ek, err := cfg.EnsureKeysFor(path, cfgDir, wd)
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("ensureKeys: %v", err))
		return false
	}
	formatRewrite, err := check.FormatWouldRewrite(raw, cfg.KeyOrder, sm.Skip, sm.Overrides, ek.Skip, ek.Keys, cfg.PinDependencyVersionsEnabled())
	if err != nil {
		ui.Row(false, path, time.Since(start), fmt.Sprintf("format: %v", err))
		return false
	}

	specific := len(scriptDrifts) + len(pinDrifts)
	if specific == 0 && !formatRewrite {
		ui.Row(true, path, time.Since(start), "")
		return true
	}
	n := specific
	if formatRewrite && specific == 0 {
		n = 1
	}
	ui.Row(false, path, time.Since(start), fmt.Sprintf("%d issue(s)", n))
	writeCheckDetails(os.Stderr, scriptDrifts, pinDrifts, formatRewrite && specific == 0)
	return false
}

func writeCheckDetails(w io.Writer, scriptDrifts []check.ScriptDrift, pinDrifts []check.PinDrift, formatOnly bool) {
	for _, d := range scriptDrifts {
		fmt.Fprintf(w, "    script %q overrides package.json - change scripts (or scriptsFiles) in pjf.yml, not here\n", d.Name)
		fmt.Fprintf(w, "      package.json: %s\n", d.Got)
		fmt.Fprintf(w, "      pjf.yml:      %s\n", d.Want)
	}
	for _, d := range pinDrifts {
		fmt.Fprintf(w, "    %s[%q]: %q would become %q - set pinDependencyVersions: false in pjf.yml or edit versions where you manage deps\n",
			d.Section, d.Package, d.Have, d.Pinned)
	}
	if formatOnly {
		fmt.Fprintln(w, "    file differs from pjf output (key order, nested sorting, trailing newline) - use keyOrder and other pjf.yml rules, or run pjf -w to apply")
	}
}
