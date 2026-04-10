package format

import (
	"strings"
	"testing"
)

func TestPinDependencyVersionString(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"^1.2.3", "1.2.3"},
		{"~2.0.0", "2.0.0"},
		{"1.0.0", "1.0.0"},
		{"workspace:*", "workspace:*"},
		{"file:../foo", "file:../foo"},
		{"npm:pkg@^1.0.0", "npm:pkg@^1.0.0"},
	}
	for _, tt := range tests {
		if got := PinDependencyVersionString(tt.in); got != tt.want {
			t.Errorf("%q: got %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormat_PinDependencyVersions(t *testing.T) {
	raw := []byte(`{"name":"x","dependencies":{"a":"^1.2.3","b":"~2.0.0"}}`)
	out, err := Format(raw, nil, false, nil, false, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "x",
  "dependencies": {
    "a": "1.2.3",
    "b": "2.0.0"
  }
}
`
	if string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormat_PinDependencyVersionsDisabled(t *testing.T) {
	raw := []byte(`{"name":"x","dependencies":{"a":"^1.2.3"}}`)
	out, err := Format(raw, nil, false, nil, false, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"^1.2.3"`) {
		t.Fatalf("expected range kept: %s", out)
	}
}
