package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestConfigGetCmd(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		config     string
		env        map[string]string
		args       []string
		wantOut    []string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "get_masks_token",
			config:  "api_url: https://a.com\nchecker_token: secret123\nteam_uid: t1\nservice_uid: s1\n",
			args:    []string{"config", "get"},
			wantOut: []string{"Config file:", "api_url: https://a.com", "checker_token: ***", "team_uid: t1", "service_uid: s1"},
		},
		{
			name:    "get_show_token",
			config:  "api_url: https://b.com\nchecker_token: visible\n",
			args:    []string{"config", "get", "--show-token"},
			wantOut: []string{"api_url: https://b.com", "checker_token: visible"},
		},
		{
			name:   "get_env_only",
			env:    map[string]string{"BITERRA_API_URL": "https://env.com", "BITERRA_CHECKER_TOKEN": "envtok"},
			args:   []string{"config", "get"},
			wantOut: []string{"Config file: (env only)", "api_url: https://env.com", "checker_token: ***"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configShowToken = false // reset flag between tests
			_ = os.Remove(".biterra.yaml")
			for _, k := range []string{"BITERRA_API_URL", "BITERRA_CHECKER_TOKEN", "BITERRA_TEAM_UID", "BITERRA_SERVICE_UID"} {
				_ = os.Unsetenv(k)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.config != "" {
				if err := os.WriteFile(".biterra.yaml", []byte(tt.config), 0600); err != nil {
					t.Fatal(err)
				}
			}
			rootCmd.SetArgs(tt.args)
			var out string
			var runErr error
			out = captureStdout(t, func() {
				runErr = rootCmd.Execute()
			})
			err := runErr
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrMsg != "" && err != nil && !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error = %v, want substring %q", err.Error(), tt.wantErrMsg)
			}
			if !tt.wantErr {
				for _, sub := range tt.wantOut {
					if !strings.Contains(out, sub) {
						t.Errorf("output %q missing %q", out, sub)
					}
				}
			}
		})
	}
}

func TestConfigSetCmd(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantErrMsg string
		thenGet    []string // substrings expected from config get after set
	}{
		{
			name:    "set_then_get",
			args:    []string{"config", "set", "--api-url", "https://set.com", "--token", "set-token", "--team-uid", "team-x", "--service-uid", "svc-y"},
			thenGet: []string{"api_url: https://set.com", "team_uid: team-x", "service_uid: svc-y"},
		},
		{
			name:    "set_minimal",
			args:    []string{"config", "set", "--api-url", "https://m.com", "--token", "t"},
			thenGet: []string{"api_url: https://m.com", "team_uid: ", "service_uid: "},
		},
		// set_missing_url / set_missing_token are not tested here because config may be
		// loaded from ~/.config/biterra/config.yaml, making the outcome environment-dependent.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configSetAPIURL, configSetToken, configSetTeam, configSetSvc = "", "", "", "" // reset flags
			_ = os.Remove(".biterra.yaml")
			for _, k := range []string{"BITERRA_API_URL", "BITERRA_CHECKER_TOKEN", "BITERRA_TEAM_UID", "BITERRA_SERVICE_UID"} {
				_ = os.Unsetenv(k)
			}
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.wantErrMsg != "" && err != nil && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error = %v, want %q", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if len(tt.thenGet) > 0 {
				configShowToken = false
				rootCmd.SetArgs([]string{"config", "get"})
				var out string
				out = captureStdout(t, func() {
					_ = rootCmd.Execute()
				})
				for _, sub := range tt.thenGet {
					if !strings.Contains(out, sub) {
						t.Errorf("after set, config get output %q missing %q", out, sub)
					}
				}
				// Token should be masked (not shown) unless --show-token
				if strings.Contains(out, "set-token") {
					t.Errorf("config get should mask token, got %q", out)
				}
			}
		})
	}
}
