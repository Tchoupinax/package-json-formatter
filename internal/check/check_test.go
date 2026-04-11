package check

import (
	"testing"
)

func TestScriptOverrideDrifts(t *testing.T) {
	raw := []byte(`{"scripts":{"build":"local","test":"vitest"}}`)
	drifts, err := ScriptOverrideDrifts(raw, false, map[string]string{
		"build": "from-yml",
		"test":  "vitest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 1 || drifts[0].Name != "build" || drifts[0].Got != "local" || drifts[0].Want != "from-yml" {
		t.Fatalf("got %+v", drifts)
	}

	drifts, err = ScriptOverrideDrifts(raw, true, map[string]string{"build": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 0 {
		t.Fatalf("skipScripts: got %+v", drifts)
	}
}

func TestFormatWouldRewrite_KeyOrder(t *testing.T) {
	raw := []byte(`{"z":"last","a":"first"}`)
	rewrite, err := FormatWouldRewrite(raw, []string{"z", "a"}, false, nil, true, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !rewrite {
		t.Fatal("expected key order fix to require rewrite")
	}
	rewrite, err = FormatWouldRewrite(raw, nil, false, nil, true, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !rewrite {
		t.Fatal("expected default key order to differ from input order")
	}
}

func TestPinDrifts(t *testing.T) {
	raw := []byte(`{"dependencies":{"a":"^1.0.0"},"devDependencies":{"b":"~2.0.0"}}`)
	drifts, err := PinDrifts(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 2 {
		t.Fatalf("got %+v", drifts)
	}
	drifts, err = PinDrifts(raw, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 0 {
		t.Fatalf("pin off: got %+v", drifts)
	}
}
