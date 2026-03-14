package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCheckCmd(t *testing.T) {
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
		handler    http.HandlerFunc
		wantOut    string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "no_config",
			wantErr:    true,
			wantErrMsg: "no config",
		},
		{
			name:   "success_with_round",
			config: "api_url: https://x.com\nchecker_token: tok\n",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"success":true,"data":{"round":{"uid":"r1","round_index":3,"started_at":"2024-01-01T00:00:00Z","ended_at":null},"tick_interval_seconds":45}}`))
			},
			wantOut: "3",
		},
		{
			name:   "success_no_round",
			config: "api_url: https://x.com\nchecker_token: tok\n",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"success":true,"data":{"round":null}}`))
			},
			wantOut: "valid",
		},
		{
			name:   "unauthorized",
			config: "api_url: https://x.com\nchecker_token: bad\n",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
			},
			wantErr:    true,
			wantErrMsg: "invalid or expired",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(".biterra.yaml")
			for _, k := range []string{"BITERRA_API_URL", "BITERRA_CHECKER_TOKEN"} {
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

			var srv *httptest.Server
			if tt.handler != nil {
				srv = httptest.NewServer(tt.handler)
				defer srv.Close()
				// Override config to point to test server
				if err := os.WriteFile(".biterra.yaml", []byte("api_url: "+srv.URL+"\nchecker_token: tok\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}

			rootCmd.SetArgs([]string{"check"})
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
			if tt.wantOut != "" && !strings.Contains(out, tt.wantOut) {
				t.Errorf("output %q missing %q", out, tt.wantOut)
			}
			if tt.name == "success_with_round" && !strings.Contains(out, "45s") {
				t.Errorf("output %q missing tick interval", out)
			}
			_ = srv
		})
	}
}
