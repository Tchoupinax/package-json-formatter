package main

import (
	"fmt"
	"os"
	"runtime"
)

// Set at link time, e.g. -ldflags "-X main.version=1.2.3 -X main.commit=... -X main.buildDate=..."
//
//nolint:gochecknoglobals // -X linker symbols must be package-level vars.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func printVersion() {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "pjf (package-json-formatter)")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "version:    ", version)
	fmt.Fprintln(os.Stdout, "commit:     ", commit)
	fmt.Fprintln(os.Stdout, "build date: ", buildDate)
	fmt.Fprintln(os.Stdout, "go:         ", runtime.Version())
	fmt.Fprintln(os.Stdout)
}
