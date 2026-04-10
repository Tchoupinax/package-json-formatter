package format

import (
	"strings"
	"testing"
)

func TestFormat_KeyOrderAndScripts(t *testing.T) {
	raw := []byte(`{"version":"1.0.0","name":"a","scripts":{"test":"jest"}}`)
	out, err := Format(raw, []string{"name", "version"}, false, map[string]string{"lint": "eslint ."}, false, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "a",
  "version": "1.0.0",
  "scripts": {
    "lint": "eslint .",
    "test": "jest"
  }
}
`
	if string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormat_SortsDependencies(t *testing.T) {
	raw := []byte(`{"name":"x","dependencies":{"z":"1","a":"2"}}`)
	out, err := Format(raw, nil, false, nil, false, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "x",
  "dependencies": {
    "a": "2",
    "z": "1"
  }
}
`
	if string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormat_ExportsNestedIndent(t *testing.T) {
	raw := []byte(`{"name":"p","exports":{".":{"import":"./a.mjs","require":"./b.js","types":"./c.d.ts"}}}`)
	out, err := Format(raw, nil, false, nil, false, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "p",
  "exports": {
    ".": {
      "import": "./a.mjs",
      "require": "./b.js",
      "types": "./c.d.ts"
    }
  }
}
`
	if string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormat_ScriptsPreMainPostOrder(t *testing.T) {
	raw := []byte(`{"name":"x","scripts":{"postbuild":"echo post","build":"tsc","prebuild":"echo pre"}}`)
	out, err := Format(raw, nil, false, nil, false, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "x",
  "scripts": {
    "prebuild": "echo pre",
    "build": "tsc",
    "postbuild": "echo post"
  }
}
`
	if string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormat_EnsureKeysOnlyIfMissing(t *testing.T) {
	raw := []byte(`{"name":"x"}`)
	ensure := map[string]interface{}{"prettier": true, "name": "ignored"}
	out, err := Format(raw, []string{"name", "prettier"}, false, nil, false, ensure, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "x",
  "prettier": true
}
`
	if string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFormat_DoesNotEscapeHTMLInStrings(t *testing.T) {
	raw := []byte(`{"name":"x","scripts":{"pipe":"node run.js < input.txt"}}`)
	out, err := Format(raw, nil, false, nil, false, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, `\u003c`) || strings.Contains(s, `\u003e`) || strings.Contains(s, `\u0026`) {
		t.Fatalf("unexpected unicode escapes in output:\n%s", out)
	}
	if !strings.Contains(s, `"pipe": "node run.js < input.txt"`) {
		t.Fatalf("expected literal < in script; got:\n%s", out)
	}
}
