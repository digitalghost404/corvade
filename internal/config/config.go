package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type CostOverride struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
}

type Config struct {
	Port           int                      `yaml:"port"`
	DashboardPort  int                      `yaml:"dashboard_port"`
	RetentionDays  int                      `yaml:"retention_days"`
	TimingGapMS    int                      `yaml:"timing_gap_ms"`
	CostOverrides  map[string]CostOverride  `yaml:"cost_overrides,omitempty"`
}

func Default() Config {
	return Config{
		Port:          4400,
		DashboardPort: 4401,
		RetentionDays: 30,
		TimingGapMS:   2000,
	}
}

func Load(path string) (Config, bool, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return cfg, false, fmt.Errorf("creating config directory: %w", err)
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return cfg, false, fmt.Errorf("marshaling default config: %w", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return cfg, false, fmt.Errorf("writing default config: %w", err)
		}
		return cfg, true, nil
	}
	if err != nil {
		return cfg, false, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, false, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, false, nil
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".corvade", "config.yaml")
}
