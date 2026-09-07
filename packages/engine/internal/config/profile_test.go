package config_test

import (
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/config"
)

func noEnv(string) string { return "" }

func TestEnvironmentSelectsTheProfile(t *testing.T) {
	env := func(k string) string {
		if k == "BURROW_PROFILE" {
			return "work"
		}
		return ""
	}
	p, err := config.ResolveProfile("", env, "/home/u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "work" {
		t.Fatalf("Name = %q, want work", p.Name)
	}
}

func TestFlagBeatsEnvironment(t *testing.T) {
	env := func(k string) string {
		if k == "BURROW_PROFILE" {
			return "work"
		}
		return ""
	}
	p, err := config.ResolveProfile("personal", env, "/home/u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "personal" {
		t.Fatalf("Name = %q, want personal", p.Name)
	}
}
