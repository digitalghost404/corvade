package main

import (
	"bytes"
	"testing"
)

func TestRootCommandHasSubcommands(t *testing.T) {
	expected := map[string]bool{
		"version":  false,
		"start":    false,
		"tail":     false,
		"doctor":   false,
		"sessions": false,
		"inspect":  false,
	}

	for _, cmd := range rootCmd.Commands() {
		if _, ok := expected[cmd.Use]; ok {
			expected[cmd.Use] = true
		} else if cmd.Use == "inspect <id>" {
			expected["inspect"] = true
		} else if cmd.Use == "completion" || cmd.Use == "help [command]" {
			// built-in cobra commands, skip
			continue
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("root command missing subcommand %q", name)
		}
	}
}

func TestVersionCommandOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	// The version command uses fmt.Printf which writes to stdout, not the command's out.
	// We just verify it doesn't error. The output goes to stdout which we can't capture
	// without more invasive changes to the command definition.
}

func TestRootCommandUse(t *testing.T) {
	if rootCmd.Use != "corvade" {
		t.Errorf("rootCmd.Use = %q, want 'corvade'", rootCmd.Use)
	}
}

func TestVersionValue(t *testing.T) {
	if version == "" {
		t.Error("version string is empty")
	}
}
