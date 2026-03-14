package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout runs fn and returns what was written to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestEnvCmd(t *testing.T) {
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
		wantOut    []string // substrings that must appear in output
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "shell_format_full",
			config:  "api_url: https://a.com\nchecker_token: t1\nteam_uid: team-1\nservice_uid: svc-1\n",
			args:    []string{"env", "--format", "shell"},
			wantOut: []string{"export BITERRA_API_URL=", "https://a.com", "export BITERRA_CHECKER_TOKEN=", "export BITERRA_TEAM_UID=", "team-1", "export BITERRA_SERVICE_UID=", "svc-1"},
		},
		{
			name:    "shell_format_minimal",
			config:  "api_url: https://b.com\nchecker_token: t2\n",
			args:    []string{"env", "--format", "shell"},
			wantOut: []string{"export BITERRA_API_URL=", "https://b.com", "export BITERRA_CHECKER_TOKEN="},
		},
		{
			name:    "dotenv_format",
			config:  "api_url: https://c.com\nchecker_token: t3\n",
			args:    []string{"env", "--format", "dotenv"},
			wantOut: []string{"BITERRA_API_URL=https://c.com", "BITERRA_CHECKER_TOKEN=t3"},
		},
		{
			name:    "exports_probe_fields",
			config:  "api_url: https://c.com\nchecker_token: t3\nprobe_type: grpc\nprobe_grpc_address: 127.0.0.1:50051\nprobe_grpc_service: world.Health\n",
			args:    []string{"env", "--format", "dotenv"},
			wantOut: []string{"BITERRA_PROBE_TYPE=grpc", "BITERRA_PROBE_GRPC_ADDRESS=127.0.0.1:50051", "BITERRA_PROBE_GRPC_SERVICE=world.Health"},
		},
		{
			name:       "unknown_format",
			config:     "api_url: https://x.com\nchecker_token: t\n",
			args:       []string{"env", "--format", "invalid"},
			wantErr:    true,
			wantErrMsg: "unknown format",
		},
		{
			name:       "no_config",
			args:       []string{"env"},
			wantErr:    true,
			wantErrMsg: "no config",
		},
		{
			name:    "env_only_config",
			env:     map[string]string{"BITERRA_API_URL": "https://env.com", "BITERRA_CHECKER_TOKEN": "env-tok"},
			args:    []string{"env", "--format", "dotenv"},
			wantOut: []string{"BITERRA_API_URL=https://env.com", "BITERRA_CHECKER_TOKEN=env-tok"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(".biterra.yaml")
			for _, k := range []string{"BITERRA_API_URL", "BITERRA_CHECKER_TOKEN", "BITERRA_TEAM_UID", "BITERRA_SERVICE_UID", "BITERRA_PROBE_TYPE", "BITERRA_PROBE_WEB_URL", "BITERRA_PROBE_BINARY_FLAG_FILE", "BITERRA_PROBE_TCP_ADDRESS", "BITERRA_PROBE_COMMAND", "BITERRA_PROBE_GRPC_ADDRESS", "BITERRA_PROBE_GRPC_SERVICE"} {
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
			if tt.wantErr {
				if tt.wantErrMsg != "" && err != nil && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error = %v, want substring %q", err.Error(), tt.wantErrMsg)
				}
				return
			}
			for _, sub := range tt.wantOut {
				if !strings.Contains(out, sub) {
					t.Errorf("output %q missing substring %q", out, sub)
				}
			}
		})
	}
}

func TestEnvCmd_OutputFormats(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	_ = os.Chdir(tmp)
	cfgPath := filepath.Join(tmp, ".biterra.yaml")
	if err := os.WriteFile(cfgPath, []byte("api_url: https://x.com\nchecker_token: secret\n"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		format string
		want   string
	}{
		{"shell", "export BITERRA_API_URL=\"https://x.com\"\n"},
		{"dotenv", "BITERRA_API_URL=https://x.com\n"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			rootCmd.SetArgs([]string{"env", "--format", tt.format})
			var out string
			out = captureStdout(t, func() {
				_ = rootCmd.Execute()
			})
			if !strings.Contains(out, tt.want) {
				t.Errorf("output %q missing %q", out, tt.want)
			}
		})
	}
}
