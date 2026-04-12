package check

import (
	"bytes"
	"encoding/json"
	"sort"

	"package-json-formatter/internal/format"
)

// FormatWouldRewrite reports whether on-disk bytes differ from format.Format output (key order,
// nested sorting, script merge, ensureKeys, pinning, trailing newline).
func FormatWouldRewrite(raw []byte, keyOrder []string, skipScripts bool, scriptOverrides map[string]string, skipEnsure bool, ensureKeys map[string]interface{}, pin bool) (bool, error) {
	out, err := format.Format(raw, keyOrder, skipScripts, scriptOverrides, skipEnsure, ensureKeys, pin)
	if err != nil {
		return false, err
	}
	return !bytesEqualNormalized(raw, out), nil
}

func bytesEqualNormalized(a, b []byte) bool {
	a = bytes.ReplaceAll(a, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	a = bytes.TrimRight(a, "\n")
	b = bytes.TrimRight(b, "\n")
	return bytes.Equal(a, b)
}

// ScriptDrift is a script name where package.json disagrees with merged pjf.yml script overrides.
type ScriptDrift struct {
	Name string
	Got  string // value in package.json
	Want string // value from config (what a format run would apply)
}

// ScriptOverrideDrifts reports script entries that the formatter would replace from config.
// When skipScripts is true (scriptsIgnore), the formatter leaves scripts untouched — no drifts.
func ScriptOverrideDrifts(raw []byte, skipScripts bool, overrides map[string]string) ([]ScriptDrift, error) {
	if skipScripts || len(overrides) == 0 {
		return nil, nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	var scripts map[string]string
	if s, ok := top["scripts"]; ok && len(s) > 0 {
		_ = json.Unmarshal(s, &scripts)
	}
	if scripts == nil {
		scripts = make(map[string]string)
	}
	var out []ScriptDrift
	for name, want := range overrides {
		got, ok := scripts[name]
		if !ok {
			continue
		}
		if got != want {
			out = append(out, ScriptDrift{Name: name, Got: got, Want: want})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// PinDrift is a dependency entry whose version string would change when pinning is enabled.
type PinDrift struct {
	Section string
	Package string
	Have    string
	Pinned  string
}

//nolint:gochecknoglobals
var dependencySections = []string{
	"dependencies",
	"devDependencies",
	"peerDependencies",
	"optionalDependencies",
}

// PinDrifts reports dependency version strings that would be rewritten by pinDependencyVersions (default on).
func PinDrifts(raw []byte, pinEnabled bool) ([]PinDrift, error) {
	if !pinEnabled {
		return nil, nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	var out []PinDrift
	for _, section := range dependencySections {
		rawMap, ok := top[section]
		if !ok || len(rawMap) == 0 {
			continue
		}
		var m map[string]string
		if err := json.Unmarshal(rawMap, &m); err != nil {
			continue
		}
		for pkg, have := range m {
			pinned := format.PinDependencyVersionString(have)
			if pinned != have {
				out = append(out, PinDrift{
					Section: section,
					Package: pkg,
					Have:    have,
					Pinned:  pinned,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Section != out[j].Section {
			return out[i].Section < out[j].Section
		}
		return out[i].Package < out[j].Package
	})
	return out, nil
}
