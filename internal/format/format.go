package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"
)

// defaultKeyOrder follows common npm package.json conventions.
func defaultKeyOrder() []string {
	return []string{
		"name",
		"version",
		"private",
		"description",
		"keywords",
		"homepage",
		"bugs",
		"license",
		"author",
		"contributors",
		"funding",
		"files",
		"main",
		"browser",
		"bin",
		"man",
		"directories",
		"repository",
		"type",
		"types",
		"typings",
		"exports",
		"module",
		"sideEffects",
		"workspaces",
		"scripts",
		"config",
		"dependencies",
		"devDependencies",
		"peerDependencies",
		"optionalDependencies",
		"bundledDependencies",
		"engines",
		"os",
		"cpu",
		"publishConfig",
	}
}

// Format returns formatted JSON with ordered top-level keys and sorted nested object keys.
// If skipScripts is true, the "scripts" field is left unchanged from the input.
// Otherwise scriptOverrides are merged into "scripts" (same name overwrites).
// If skipEnsure is false and ensureKeys is non-empty, each listed top-level key is set only when missing.
// If pinDependencyVersions is true, leading ^ and ~ are removed from versions in dependencies, devDependencies,
// peerDependencies, and optionalDependencies.
func Format(raw []byte, keyOrder []string, skipScripts bool, scriptOverrides map[string]string, skipEnsure bool, ensureKeys map[string]interface{}, pinDependencyVersions bool) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	order := keyOrder
	if len(order) == 0 {
		order = defaultKeyOrder()
	}

	if !skipScripts && len(scriptOverrides) > 0 {
		mergeScripts(top, scriptOverrides)
	}

	if !skipEnsure && len(ensureKeys) > 0 {
		ensureMissingKeys(top, ensureKeys)
	}

	if pinDependencyVersions {
		pinDependencyMaps(top)
	}

	buf := new(bytes.Buffer)
	if err := writeOrderedObject(buf, top, order, 0); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func mergeScripts(top map[string]json.RawMessage, overrides map[string]string) {
	var scripts map[string]string
	if raw, ok := top["scripts"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &scripts)
	}
	if scripts == nil {
		scripts = make(map[string]string)
	}
	maps.Copy(scripts, overrides)
	b, err := marshalJSON(scripts)
	if err != nil {
		return
	}
	top["scripts"] = b
}

func ensureMissingKeys(top map[string]json.RawMessage, ensure map[string]interface{}) {
	for k, v := range ensure {
		if k == "" {
			continue
		}
		if _, ok := top[k]; ok {
			continue
		}
		raw, err := marshalJSON(v)
		if err != nil {
			continue
		}
		top[k] = raw
	}
}

func writeOrderedObject(w *bytes.Buffer, obj map[string]json.RawMessage, preferredOrder []string, depth int) error {
	w.WriteByte('{')
	if len(obj) == 0 {
		w.WriteByte('}')
		return nil
	}

	ordered := orderKeys(obj, preferredOrder)
	first := true
	indent := bytes.Repeat([]byte("  "), depth+1)
	closeIndent := bytes.Repeat([]byte("  "), depth)

	for _, k := range ordered {
		if !first {
			w.WriteByte(',')
		}
		first = false
		w.WriteByte('\n')
		w.Write(indent)
		keyJSON, err := marshalJSON(k)
		if err != nil {
			return err
		}
		w.Write(keyJSON)
		w.WriteString(": ")

		raw := obj[k]
		if err := writeValue(w, k, raw, depth+1); err != nil {
			return err
		}
	}
	w.WriteByte('\n')
	w.Write(closeIndent)
	w.WriteByte('}')
	return nil
}

func orderKeys(obj map[string]json.RawMessage, preferred []string) []string {
	prefSet := make(map[string]int)
	for i, k := range preferred {
		prefSet[k] = i
	}
	var rest []string
	for k := range obj {
		if _, ok := prefSet[k]; !ok {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)

	var out []string
	seen := make(map[string]bool)
	for _, k := range preferred {
		if _, ok := obj[k]; ok && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	out = append(out, rest...)
	return out
}

func writeValue(w *bytes.Buffer, key string, raw json.RawMessage, depth int) error {
	if len(raw) == 0 {
		w.WriteString("null")
		return nil
	}

	switch key {
	case "scripts":
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return writeIndentedRaw(w, raw, depth)
		}
		return writeScriptsMap(w, m, depth)
	case "dependencies", "devDependencies", "peerDependencies",
		"optionalDependencies", "bundledDependencies", "engines", "os", "cpu":
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return writeIndentedRaw(w, raw, depth)
		}
		return writeSortedStringMap(w, m, depth)

	case "bugs":
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return writeIndentedRaw(w, raw, depth)
		}
		return writeSortedStringMap(w, m, depth)

	case "repository":
		// string or object
		if raw[0] == '"' {
			return writeIndentedRaw(w, raw, depth)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return writeIndentedRaw(w, raw, depth)
		}
		return writeSortedStringMap(w, m, depth)

	case "exports":
		return writeExports(w, raw, depth)
	}

	// Arrays: workspaces, files, keywords, etc. -- pretty-print with sorted strings where obvious
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return writeIndentedRaw(w, raw, depth)
	}
	return writeGeneric(w, generic, depth)
}

// scriptGroup orders npm scripts: pre<stem>, <stem>, post<stem> for the same stem.
// prepare and prettier are treated as single names (not pre+<rest>).
func scriptGroup(k string) (group string, tier int) {
	switch {
	case k == "prepare":
		return "prepare", 1
	case k == "prettier":
		return "prettier", 1
	case strings.HasPrefix(k, "post") && len(k) > 4:
		return k[4:], 2
	case strings.HasPrefix(k, "pre") && len(k) > 3:
		return k[3:], 0
	default:
		return k, 1
	}
}

func sortScriptKeys(keys []string) {
	sort.SliceStable(keys, func(i, j int) bool {
		gi, ti := scriptGroup(keys[i])
		gj, tj := scriptGroup(keys[j])
		if gi != gj {
			return gi < gj
		}
		if ti != tj {
			return ti < tj
		}
		return keys[i] < keys[j]
	})
}

func writeSortedStringMap(w *bytes.Buffer, m map[string]json.RawMessage, depth int) error {
	return writeOrderedStringMap(w, m, depth, sort.Strings)
}

func writeScriptsMap(w *bytes.Buffer, m map[string]json.RawMessage, depth int) error {
	return writeOrderedStringMap(w, m, depth, sortScriptKeys)
}

func writeOrderedStringMap(w *bytes.Buffer, m map[string]json.RawMessage, depth int, orderKeys func([]string)) error {
	if len(m) == 0 {
		w.WriteString("{}")
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	orderKeys(keys)
	w.WriteByte('{')
	indent := bytes.Repeat([]byte("  "), depth+1)
	closeIndent := bytes.Repeat([]byte("  "), depth)
	first := true
	for _, k := range keys {
		if !first {
			w.WriteByte(',')
		}
		first = false
		w.WriteByte('\n')
		w.Write(indent)
		keyJSON, err := marshalJSON(k)
		if err != nil {
			return err
		}
		w.Write(keyJSON)
		w.WriteString(": ")
		if err := writeIndentedRaw(w, m[k], depth); err != nil {
			return err
		}
	}
	w.WriteByte('\n')
	w.Write(closeIndent)
	w.WriteByte('}')
	return nil
}

func writeExports(w *bytes.Buffer, raw json.RawMessage, depth int) error {
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asMap); err == nil && len(asMap) > 0 {
		return writeSortedStringMap(w, asMap, depth)
	}
	var asStr string
	if err := json.Unmarshal(raw, &asStr); err == nil {
		return writeIndentedRaw(w, raw, depth)
	}
	return writeIndentedRaw(w, raw, depth)
}

func writeGeneric(w *bytes.Buffer, v interface{}, depth int) error {
	switch t := v.(type) {
	case map[string]interface{}:
		rm := make(map[string]json.RawMessage)
		for k, val := range t {
			b, err := marshalJSON(val)
			if err != nil {
				return err
			}
			rm[k] = b
		}
		keys := make([]string, 0, len(rm))
		for k := range rm {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return writeOrderedObject(w, rm, keys, depth)
	case []interface{}:
		return writeArray(w, t, depth)
	default:
		b, err := marshalJSON(v)
		if err != nil {
			return err
		}
		return writeIndentedRaw(w, b, depth)
	}
}

func writeArray(w *bytes.Buffer, arr []interface{}, depth int) error {
	w.WriteByte('[')
	if len(arr) == 0 {
		w.WriteByte(']')
		return nil
	}
	indent := bytes.Repeat([]byte("  "), depth+1)
	closeIndent := bytes.Repeat([]byte("  "), depth)
	for i, el := range arr {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteByte('\n')
		w.Write(indent)
		if err := writeGeneric(w, el, depth+1); err != nil {
			return err
		}
	}
	w.WriteByte('\n')
	w.Write(closeIndent)
	w.WriteByte(']')
	return nil
}

// depth is the nesting level of the map that contains this value (same as writeOrderedStringMap).
func writeIndentedRaw(w *bytes.Buffer, raw []byte, depth int) error {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		w.Write(raw)
		return nil //nolint:nilerr // invalid JSON: emit raw fragment unchanged
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		w.Write(raw)
		return nil //nolint:nilerr // re-indent failed: emit raw fragment unchanged
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	s := string(b)
	lines := strings.Split(s, "\n")
	prefix := strings.Repeat("  ", depth+1)
	w.WriteString(lines[0])
	for i := 1; i < len(lines); i++ {
		w.WriteByte('\n')
		w.WriteString(prefix)
		w.WriteString(lines[i])
	}
	return nil
}

// marshalJSON matches encoding/json.Marshal but does not escape <, >, & in strings
// (Go's default breaks shell redirection and URLs in package.json scripts).
func marshalJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}
