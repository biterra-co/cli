package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		file           string
		content        string
		env            map[string]string
		wantAPIURL     string
		wantToken      string
		wantTeam       string
		wantService    string
		wantPathPrefix string
		wantErr        bool
	}{
		{
			name:           "no_file_env_only",
			env:            map[string]string{"BITERRA_API_URL": "https://api.example.com", "BITERRA_CHECKER_TOKEN": "secret"},
			wantAPIURL:     "https://api.example.com",
			wantToken:      "secret",
			wantPathPrefix: "",
		},
		{
			name:           "yaml_file",
			file:           ".biterra.yaml",
			content:        "api_url: https://file.example.com\nchecker_token: file-token\nteam_uid: team-1\nservice_uid: svc-1\n",
			wantAPIURL:     "https://file.example.com",
			wantToken:      "file-token",
			wantTeam:       "team-1",
			wantService:    "svc-1",
			wantPathPrefix: ".biterra.yaml",
		},
		{
			name:           "env_overrides_file",
			file:           ".biterra.yaml",
			content:        "api_url: https://file.example.com\nchecker_token: file-token\n",
			env:            map[string]string{"BITERRA_API_URL": "https://env.example.com", "BITERRA_CHECKER_TOKEN": "env-token"},
			wantAPIURL:     "https://env.example.com",
			wantToken:      "env-token",
			wantPathPrefix: ".biterra.yaml",
		},
		{
			name:           "partial_env_override",
			file:           ".biterra.yaml",
			content:        "api_url: https://file.example.com\nchecker_token: file-token\n",
			env:            map[string]string{"BITERRA_API_URL": "https://env.example.com"},
			wantAPIURL:     "https://env.example.com",
			wantToken:      "file-token",
			wantPathPrefix: ".biterra.yaml",
		},
		{
			name:    "invalid_yaml",
			file:    ".biterra.yaml",
			content: "api_url: [invalid\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset env for this test
			for _, k := range []string{"BITERRA_API_URL", "BITERRA_CHECKER_TOKEN", "BITERRA_TEAM_UID", "BITERRA_SERVICE_UID", "BITERRA_PROBE_TYPE", "BITERRA_PROBE_WEB_URL", "BITERRA_PROBE_BINARY_FLAG_FILE"} {
				_ = os.Unsetenv(k)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.file != "" {
				if err := os.WriteFile(tt.file, []byte(tt.content), 0600); err != nil {
					t.Fatal(err)
				}
			}

			cfg, path, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cfg == nil {
				t.Fatal("Load() returned nil config")
			}
			if cfg.APIURL != tt.wantAPIURL {
				t.Errorf("APIURL = %q, want %q", cfg.APIURL, tt.wantAPIURL)
			}
			if cfg.CheckerToken != tt.wantToken {
				t.Errorf("CheckerToken = %q, want %q", cfg.CheckerToken, tt.wantToken)
			}
			if cfg.TeamUID != tt.wantTeam {
				t.Errorf("TeamUID = %q, want %q", cfg.TeamUID, tt.wantTeam)
			}
			if cfg.ServiceUID != tt.wantService {
				t.Errorf("ServiceUID = %q, want %q", cfg.ServiceUID, tt.wantService)
			}
			if tt.wantPathPrefix != "" && path != tt.wantPathPrefix {
				// path might be ./file or full path depending on cwd
				if filepath.Base(path) != filepath.Base(tt.wantPathPrefix) {
					t.Errorf("path = %q, want prefix/base %q", path, tt.wantPathPrefix)
				}
			}
		})
	}
}

func TestLoadRequired(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	_ = os.Chdir(tmp)

	tests := []struct {
		name      string
		file      string
		content   string
		env       map[string]string
		wantErr   bool
		wantExist bool // true = expect os.ErrNotExist when wantErr
	}{
		{
			name:      "missing_both",
			wantErr:   true,
			wantExist: true,
		},
		{
			name:      "missing_token",
			env:       map[string]string{"BITERRA_API_URL": "https://x.com"},
			wantErr:   true,
			wantExist: true,
		},
		{
			name:      "missing_url",
			env:       map[string]string{"BITERRA_CHECKER_TOKEN": "t"},
			wantErr:   true,
			wantExist: true,
		},
		{
			name:    "ok_from_env",
			env:     map[string]string{"BITERRA_API_URL": "https://x.com", "BITERRA_CHECKER_TOKEN": "t"},
			wantErr: false,
		},
		{
			name:    "ok_from_file",
			file:    ".biterra.yaml",
			content: "api_url: https://f.com\nchecker_token: tok\n",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range []string{"BITERRA_API_URL", "BITERRA_CHECKER_TOKEN", "BITERRA_TEAM_UID", "BITERRA_SERVICE_UID", "BITERRA_PROBE_TYPE", "BITERRA_PROBE_WEB_URL", "BITERRA_PROBE_BINARY_FLAG_FILE"} {
				_ = os.Unsetenv(k)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.file != "" {
				_ = os.WriteFile(tt.file, []byte(tt.content), 0600)
			}

			cfg, err := LoadRequired()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRequired() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.wantExist && !os.IsNotExist(err) {
					t.Errorf("expected os.ErrNotExist, got %v", err)
				}
				return
			}
			if cfg == nil || cfg.APIURL == "" || cfg.CheckerToken == "" {
				t.Errorf("expected valid config")
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "save_local",
			cfg:  &Config{APIURL: "https://a.com", CheckerToken: "t", TeamUID: "team1", ServiceUID: "svc1"},
		},
		{
			name: "save_optional_empty",
			cfg:  &Config{APIURL: "https://b.com", CheckerToken: "t2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Save(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			// Should have created .biterra.yaml in cwd (tmp)
			data, err := os.ReadFile(".biterra.yaml")
			if err != nil {
				t.Errorf("Save() did not create .biterra.yaml: %v", err)
				return
			}
			loaded, _, err := Load()
			if err != nil {
				t.Errorf("Load after Save failed: %v", err)
				return
			}
			if loaded.APIURL != tt.cfg.APIURL || loaded.CheckerToken != tt.cfg.CheckerToken {
				t.Errorf("loaded config mismatch: got api_url=%q token=%q", loaded.APIURL, loaded.CheckerToken)
			}
			_ = data
		})
	}
}
