package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the hx configuration file.
type Config struct {
	Templates []TemplateConfig `yaml:"templates"`
}

// TemplateConfig represents a template defined in the YAML config.
type TemplateConfig struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
}

// DefaultPath returns the default config file path (~/.config/hx/config.yaml).
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hx", "config.yaml")
}

// TemplatesPath returns the templates config file path (~/.config/hx/templates.yaml).
func TemplatesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hx", "templates.yaml")
}

// LoadTemplates loads templates from the YAML config file.
// Returns an empty slice if the file doesn't exist.
func LoadTemplates(path string) ([]TemplateConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return cfg.Templates, nil
}

// SaveTemplates writes templates to the YAML config file.
func SaveTemplates(path string, templates []TemplateConfig) error {
	cfg := Config{Templates: templates}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
