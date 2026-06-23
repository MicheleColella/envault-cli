package main

import (
	"testing"
)

func TestRootVersion(t *testing.T) {
	cmd := newRootCmd("1.2.3")
	if cmd.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", cmd.Version, "1.2.3")
	}
}

func TestRootCommandsRegistered(t *testing.T) {
	cmd := newRootCmd("dev")
	want := []string{"init", "key", "import", "add", "list", "push", "pull", "run", "hook"}

	registered := make(map[string]bool)
	for _, c := range cmd.Commands() {
		registered[c.Name()] = true
	}

	for _, name := range want {
		if !registered[name] {
			t.Errorf("command %q not registered", name)
		}
	}
}
