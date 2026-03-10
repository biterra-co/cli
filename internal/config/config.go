package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds API URL, checker token, and optional team/service for the checker process.
type Config struct {
	APIURL       string `yaml:"api_url" json:"api_url"`
	CheckerToken string `yaml:"checker_token" json:"checker_token"`
	TeamUID      string `yaml:"team_uid,omitempty" json:"team_uid,omitempty"`
	ServiceUID   string `yaml:"service_uid,omitempty" json:"service_uid,omitempty"`
}

// Load reads config from project-local file, then user global, with env overrides. Returns (config, pathUsed, error).
func Load() (*Config, string, error) {
	cfg := &Config{}
	cfg.APIURL = os.Getenv("BITERRA_API_URL")
	cfg.CheckerToken = os.Getenv("BITERRA_CHECKER_TOKEN")
	cfg.TeamUID = os.Getenv("BITERRA_TEAM_UID")
	cfg.ServiceUID = os.Getenv("BITERRA_SERVICE_UID")

	// Prefer project-local then global
	localPaths := []string{"./.biterra.yaml", "./.biterra.json"}
	home, _ := os.UserHomeDir()
	if home != "" {
		localPaths = append(localPaths, filepath.Join(home, ".config", "biterra", "config.yaml"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		localPaths = append(localPaths, filepath.Join(xdg, "biterra", "config.yaml"))
	}

	var path string
	for _, p := range localPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, "", err
		}
		path = p
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, "", fmt.Errorf("%s: %w", p, err)
		}
		break
	}
	if path == "" {
		// No file found; env-only config is valid if both URL and token set
		return cfg, "", nil
	}
	// Env overrides file
	if v := os.Getenv("BITERRA_API_URL"); v != "" {
		cfg.APIURL = v
	}
	if v := os.Getenv("BITERRA_CHECKER_TOKEN"); v != "" {
		cfg.CheckerToken = v
	}
	if v := os.Getenv("BITERRA_TEAM_UID"); v != "" {
		cfg.TeamUID = v
	}
	if v := os.Getenv("BITERRA_SERVICE_UID"); v != "" {
		cfg.ServiceUID = v
	}
	return cfg, path, nil
}

// LoadRequired returns config or error if no API URL/token (from file or env).
func LoadRequired() (*Config, error) {
	cfg, _, err := Load()
	if err != nil {
		return nil, err
	}
	if cfg.APIURL == "" || cfg.CheckerToken == "" {
		return nil, os.ErrNotExist
	}
	return cfg, nil
}

// Save writes config to project-local ./.biterra.yaml if the current dir is writable, else ~/.config/biterra/config.yaml.
func Save(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	// Prefer project-local
	localPath := ".biterra.yaml"
	if err := os.WriteFile(localPath, data, 0600); err == nil {
		return nil
	}
	// Fallback: user global
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".config", "biterra")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	outPath := filepath.Join(dir, "config.yaml")
	return os.WriteFile(outPath, data, 0600)
}
