package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds API URL, checker token, optional customer portal URL, and optional team/service for the checker process.
type Config struct {
	APIURL            string `yaml:"api_url" json:"api_url"`
	CheckerToken      string `yaml:"checker_token" json:"checker_token"`
	CustomerPortalURL string `yaml:"customer_portal_url,omitempty" json:"customer_portal_url,omitempty"`
	TeamUID           string `yaml:"team_uid,omitempty" json:"team_uid,omitempty"`
	ServiceUID        string `yaml:"service_uid,omitempty" json:"service_uid,omitempty"`
	ProbeType         string `yaml:"probe_type,omitempty" json:"probe_type,omitempty"`                         // web | binary | tcp | command | grpc
	ProbeWebURL       string `yaml:"probe_web_url,omitempty" json:"probe_web_url,omitempty"`                   // used when probe_type=web
	ProbeBinaryFile   string `yaml:"probe_binary_flag_file,omitempty" json:"probe_binary_flag_file,omitempty"` // used when probe_type=binary
	ProbeTCPAddress   string `yaml:"probe_tcp_address,omitempty" json:"probe_tcp_address,omitempty"`           // used when probe_type=tcp
	ProbeCommand      string `yaml:"probe_command,omitempty" json:"probe_command,omitempty"`                   // used when probe_type=command
	ProbeGRPCAddress  string `yaml:"probe_grpc_address,omitempty" json:"probe_grpc_address,omitempty"`         // used when probe_type=grpc
	ProbeGRPCService  string `yaml:"probe_grpc_service,omitempty" json:"probe_grpc_service,omitempty"`         // optional when probe_type=grpc
}

// Load reads config from project-local file, then user global, with env overrides. Returns (config, pathUsed, error).
func Load() (*Config, string, error) {
	cfg := &Config{}
	cfg.APIURL = os.Getenv("BITERRA_API_URL")
	cfg.CheckerToken = os.Getenv("BITERRA_CHECKER_TOKEN")
	cfg.CustomerPortalURL = os.Getenv("BITERRA_CUSTOMER_PORTAL_URL")
	cfg.TeamUID = os.Getenv("BITERRA_TEAM_UID")
	cfg.ServiceUID = os.Getenv("BITERRA_SERVICE_UID")
	cfg.ProbeType = os.Getenv("BITERRA_PROBE_TYPE")
	cfg.ProbeWebURL = os.Getenv("BITERRA_PROBE_WEB_URL")
	cfg.ProbeBinaryFile = os.Getenv("BITERRA_PROBE_BINARY_FLAG_FILE")
	cfg.ProbeTCPAddress = os.Getenv("BITERRA_PROBE_TCP_ADDRESS")
	cfg.ProbeCommand = os.Getenv("BITERRA_PROBE_COMMAND")
	cfg.ProbeGRPCAddress = os.Getenv("BITERRA_PROBE_GRPC_ADDRESS")
	cfg.ProbeGRPCService = os.Getenv("BITERRA_PROBE_GRPC_SERVICE")

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
	if v := os.Getenv("BITERRA_PROBE_TYPE"); v != "" {
		cfg.ProbeType = v
	}
	if v := os.Getenv("BITERRA_PROBE_WEB_URL"); v != "" {
		cfg.ProbeWebURL = v
	}
	if v := os.Getenv("BITERRA_PROBE_BINARY_FLAG_FILE"); v != "" {
		cfg.ProbeBinaryFile = v
	}
	if v := os.Getenv("BITERRA_PROBE_TCP_ADDRESS"); v != "" {
		cfg.ProbeTCPAddress = v
	}
	if v := os.Getenv("BITERRA_PROBE_COMMAND"); v != "" {
		cfg.ProbeCommand = v
	}
	if v := os.Getenv("BITERRA_PROBE_GRPC_ADDRESS"); v != "" {
		cfg.ProbeGRPCAddress = v
	}
	if v := os.Getenv("BITERRA_PROBE_GRPC_SERVICE"); v != "" {
		cfg.ProbeGRPCService = v
	}
	if v := os.Getenv("BITERRA_CUSTOMER_PORTAL_URL"); v != "" {
		cfg.CustomerPortalURL = v
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
