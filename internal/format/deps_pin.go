package format

import (
	"encoding/json"
	"strings"
)

var dependencyMapKeys = []string{
	"dependencies",
	"devDependencies",
	"peerDependencies",
	"optionalDependencies",
}

// PinDependencyVersionString turns ^1.2.3 and ~1.2.3 into 1.2.3. Non-semver refs are unchanged.
func PinDependencyVersionString(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || isNonSemverDependencyRef(v) {
		return v
	}
	if strings.HasPrefix(v, "^") {
		return strings.TrimPrefix(v, "^")
	}
	if strings.HasPrefix(v, "~") {
		return strings.TrimPrefix(v, "~")
	}
	return v
}

func isNonSemverDependencyRef(v string) bool {
	if strings.HasPrefix(v, "workspace:") || strings.HasPrefix(v, "file:") ||
		strings.HasPrefix(v, "link:") || strings.HasPrefix(v, "npm:") {
		return true
	}
	if strings.Contains(v, "://") {
		return true
	}
	if strings.HasPrefix(v, "git+") || strings.HasPrefix(v, "git:") || strings.HasPrefix(v, "github:") {
		return true
	}
	return false
}

func pinDependencyMaps(top map[string]json.RawMessage) {
	for _, key := range dependencyMapKeys {
		raw, ok := top[key]
		if !ok || len(raw) == 0 {
			continue
		}
		if out, changed := pinDependencyMapRaw(raw); changed {
			top[key] = out
		}
	}
}

func pinDependencyMapRaw(raw json.RawMessage) (json.RawMessage, bool) {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw, false
	}
	changed := false
	for k, v := range m {
		nv := PinDependencyVersionString(v)
		if nv != v {
			m[k] = nv
			changed = true
		}
	}
	if !changed {
		return raw, false
	}
	b, err := marshalJSON(m)
	if err != nil {
		return raw, false
	}
	return b, true
}
