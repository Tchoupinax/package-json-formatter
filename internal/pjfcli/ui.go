package pjfcli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	cReset    = "\033[0m"
	cBold     = "\033[1m"
	cDim      = "\033[2m"
	cGreen    = "\033[32m"
	cRed      = "\033[31m"
	cCyan     = "\033[36m"
	cYellow        = "\033[33m"
	cFolderHighlight = "\033[1;95m" // bold bright magenta: parent dir of package.json
	pathColWidth     = 72
	lineWidth        = 4 + pathColWidth + 12
)

var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// UI prints formatted progress to stderr.
type UI struct {
	color bool
	wd    string
	w     io.Writer
}

// New builds a UI using paths relative to wd when possible.
func New(wd string) *UI {
	return &UI{
		color: stderrIsColor(),
		wd:    wd,
		w:     os.Stderr,
	}
}

func stderrIsColor() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Header prints the table title when n > 1.
func (u *UI) Header(n int) {
	if n <= 1 {
		return
	}
	fmt.Fprintln(u.w)
	title := "pjf"
	if u.color {
		fmt.Fprintf(u.w, "%s%s%s  %s%d package.json%s\n", cBold+cCyan, title, cReset, cDim, n, cReset)
	} else {
		fmt.Fprintf(u.w, "%s  (%d package.json)\n", title, n)
	}
	fmt.Fprintf(u.w, "%s\n", strings.Repeat("-", lineWidth))
	fmt.Fprintf(u.w, "%-4s %-*s %s\n", "", pathColWidth, "package.json", "time")
	fmt.Fprintf(u.w, "%s\n", strings.Repeat("-", lineWidth))
}

// Footer prints totals when n > 1.
func (u *UI) Footer(n int, okN, failN int, total time.Duration) {
	if n <= 1 {
		return
	}
	fmt.Fprintf(u.w, "%s\n", strings.Repeat("-", lineWidth))
	line := fmt.Sprintf("%d file", n)
	if n != 1 {
		line += "s"
	}
	line += fmt.Sprintf("  %d ok", okN)
	if failN > 0 {
		line += fmt.Sprintf("  %d failed", failN)
	}
	line += fmt.Sprintf("  %s total", formatHumanDur(total))
	if u.color {
		fmt.Fprintf(u.w, "%s%s%s\n", cDim, line, cReset)
	} else {
		fmt.Fprintln(u.w, line)
	}
	fmt.Fprintln(u.w)
}

// Row prints one result line.
func (u *UI) Row(ok bool, path string, d time.Duration, errDetail string) {
	rel := filepath.ToSlash(pathForDisplay(u.wd, path))
	if len(rel) > pathColWidth {
		mid := pathColWidth - 3 // "..."
		left := mid / 2
		right := mid - left
		rel = rel[:left] + "..." + rel[len(rel)-right:]
	}

	st := " ok"
	if !ok {
		st = "FAIL"
	}

	var sb strings.Builder
	if u.color {
		if ok {
			fmt.Fprintf(&sb, "%s%s%-4s%s ", cGreen, cBold, st, cReset)
		} else {
			fmt.Fprintf(&sb, "%s%s%-4s%s ", cRed, cBold, st, cReset)
		}
		pathCell := pathCellHighlighted(rel)
		fmt.Fprintf(&sb, "%s ", padVisibleRight(pathCell, pathColWidth))
		fmt.Fprintf(&sb, "%s%10s%s", cYellow, formatHumanDur(d), cReset)
	} else {
		fmt.Fprintf(&sb, "%-4s %-*s %10s", st, pathColWidth, rel, formatHumanDur(d))
	}

	if errDetail != "" {
		if u.color {
			fmt.Fprintf(&sb, "  %s%s%s", cRed, errDetail, cReset)
		} else {
			fmt.Fprintf(&sb, "  %s", errDetail)
		}
	}
	fmt.Fprintln(u.w, sb.String())
}

// pathCellHighlighted dims the path but uses cFolderHighlight on the directory name
// immediately before /package.json (e.g. .../apps/web/package.json -> "web" stands out).
func pathCellHighlighted(rel string) string {
	prefix, folder, ok := splitPackageJSONPath(rel)
	if !ok || folder == "" {
		return cDim + rel + cReset
	}
	return cDim + prefix + cFolderHighlight + folder + cReset + cDim + "/package.json" + cReset
}

func splitPackageJSONPath(rel string) (prefix, folder string, ok bool) {
	if filepath.Base(rel) != "package.json" {
		return "", "", false
	}
	dir := filepath.Dir(rel)
	if dir == "." || dir == "" {
		return "", "", false
	}
	folder = filepath.Base(dir)
	tail := folder + "/package.json"
	if !strings.HasSuffix(rel, tail) {
		return "", "", false
	}
	prefix = strings.TrimSuffix(rel, tail)
	return prefix, folder, true
}

func padVisibleRight(ansi string, width int) string {
	plain := stripANSI(ansi)
	if len(plain) >= width {
		return ansi
	}
	return ansi + strings.Repeat(" ", width-len(plain))
}

func stripANSI(s string) string {
	return reANSI.ReplaceAllString(s, "")
}

func pathForDisplay(wd, p string) string {
	p = filepath.Clean(p)
	wd = filepath.Clean(wd)
	rel, err := filepath.Rel(wd, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return rel
}

func formatHumanDur(d time.Duration) string {
	d = d.Truncate(time.Microsecond)
	if d < time.Second {
		ms := float64(d.Nanoseconds()) / 1e6
		if ms < 0.01 && d > 0 {
			return "<0.01ms"
		}
		if ms < 10 {
			return fmt.Sprintf("%.2fms", ms)
		}
		return fmt.Sprintf("%.1fms", ms)
	}
	return d.Round(time.Millisecond).String()
}
