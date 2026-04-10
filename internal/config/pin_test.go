package config

import "testing"

func TestPinDependencyVersionsEnabled_DefaultTrue(t *testing.T) {
	var c Config
	if !c.PinDependencyVersionsEnabled() {
		t.Fatal("expected default true")
	}
	f := false
	c.PinDependencyVersions = &f
	if c.PinDependencyVersionsEnabled() {
		t.Fatal("expected false when set")
	}
}
