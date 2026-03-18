package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Port != 4400 {
		t.Errorf("expected port 4400, got %d", cfg.Port)
	}
	if cfg.DashboardPort != 4401 {
		t.Errorf("expected dashboard port 4401, got %d", cfg.DashboardPort)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", cfg.RetentionDays)
	}
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfg, created, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected config file to be created")
	}
	if cfg.Port != 4400 {
		t.Errorf("expected default port 4400, got %d", cfg.Port)
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("config file was not created on disk")
	}
}

func TestLoadExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte("port: 5500\ndashboard_port: 5501\nretention_days: 7\n")
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, created, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not report created for existing file")
	}
	if cfg.Port != 5500 {
		t.Errorf("expected port 5500, got %d", cfg.Port)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("expected 7 retention days, got %d", cfg.RetentionDays)
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
	if !strings.HasSuffix(p, filepath.Join(".corvade", "config.yaml")) {
		t.Errorf("unexpected default path: %s", p)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("{{bad yaml: ["), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("expected 'parsing config' in error, got: %v", err)
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("port: 1234"), 0000); err != nil {
		t.Fatal(err)
	}

	_, _, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Errorf("expected 'reading config' in error, got: %v", err)
	}
}

func TestLoadCostOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`port: 4400
cost_overrides:
  gpt-4:
    input_per_1k: 0.03
    output_per_1k: 0.06
  claude-3:
    input_per_1k: 0.01
    output_per_1k: 0.02
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, created, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not report created")
	}
	if len(cfg.CostOverrides) != 2 {
		t.Fatalf("expected 2 cost overrides, got %d", len(cfg.CostOverrides))
	}
	gpt4 := cfg.CostOverrides["gpt-4"]
	if gpt4.InputPer1K != 0.03 {
		t.Errorf("gpt-4 input: got %f, want 0.03", gpt4.InputPer1K)
	}
	if gpt4.OutputPer1K != 0.06 {
		t.Errorf("gpt-4 output: got %f, want 0.06", gpt4.OutputPer1K)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty YAML unmarshals over defaults, defaults should remain
	if cfg.Port != 4400 {
		t.Errorf("expected default port 4400, got %d", cfg.Port)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected default retention 30, got %d", cfg.RetentionDays)
	}
}

func TestLoadPartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Only set port, other fields should keep defaults
	content := []byte("port: 9999\n")
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Port)
	}
	if cfg.DashboardPort != 4401 {
		t.Errorf("expected default dashboard port 4401, got %d", cfg.DashboardPort)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected default retention 30, got %d", cfg.RetentionDays)
	}
	if cfg.TimingGapMS != 2000 {
		t.Errorf("expected default timing gap 2000, got %d", cfg.TimingGapMS)
	}
}

func TestLoadCannotCreateDir(t *testing.T) {
	// Make a directory with no write permission, then try to create a subdir inside it
	tmpDir := t.TempDir()
	noWrite := filepath.Join(tmpDir, "nowrite")
	if err := os.MkdirAll(noWrite, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(noWrite, 0755) })
	cfgPath := filepath.Join(noWrite, "sub", "config.yaml")

	_, _, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error when directory creation fails")
	}
	if !strings.Contains(err.Error(), "creating config directory") {
		t.Errorf("expected 'creating config directory' in error, got: %v", err)
	}
}

func TestLoadCannotWriteDefault(t *testing.T) {
	// Create directory with no write permission, so WriteFile fails
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "nowrite")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(subDir, "config.yaml")
	// Remove write permission from directory
	if err := os.Chmod(subDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(subDir, 0755) })

	_, _, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error when writing default config fails")
	}
	if !strings.Contains(err.Error(), "writing default config") {
		t.Errorf("expected 'writing default config' in error, got: %v", err)
	}
}

func TestLoadCreatesDirWhenNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "sub", "dir", "config.yaml")

	cfg, created, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	if cfg.Port != 4400 {
		t.Errorf("expected default port, got %d", cfg.Port)
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}
