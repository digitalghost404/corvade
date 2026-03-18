package config

import (
	"os"
	"path/filepath"
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
