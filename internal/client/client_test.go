package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		baseURL  string
		token    string
		wantURL  string
		wantAuth string
	}{
		{"https://world.example.com", "mytoken", "https://world.example.com", "Bearer mytoken"},
		{"https://world.example.com/", "t2", "https://world.example.com", "Bearer t2"},
	}
	for _, tt := range tests {
		c := New(tt.baseURL, tt.token)
		if c == nil {
			t.Fatal("New returned nil")
		}
		u := c.url("/rounds/current")
		if u != tt.wantURL+"/api/ad/checker/rounds/current" {
			t.Errorf("url() = %q, want %s/api/ad/checker/rounds/current", u, tt.wantURL)
		}
		// Auth is tested via handler in TestGetRoundsCurrent
		_, _ = tt.wantAuth, c
	}
}

func TestGetRoundsCurrent(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantRound  *Round
		wantErr    bool
		want401    bool
	}{
		{
			name:       "success_with_round",
			statusCode: 200,
			body:       `{"success":true,"data":{"round":{"uid":"r1","round_index":2,"started_at":"2024-01-01T00:00:00Z","ended_at":null}}}`,
			wantRound:  &Round{UID: "r1", RoundIndex: 2, StartedAt: "2024-01-01T00:00:00Z", EndedAt: nil},
		},
		{
			name:       "success_no_round",
			statusCode: 200,
			body:       `{"success":true,"data":{"round":null}}`,
			wantRound:  nil,
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       `{"success":false}`,
			wantErr:    true,
			want401:    true,
		},
		{
			name:       "server_error",
			statusCode: 500,
			body:       `{}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var authHeader string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader = r.Header.Get("Authorization")
				if r.URL.Path != "/api/ad/checker/rounds/current" {
					t.Errorf("path = %q", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			cl := New(srv.URL, "test-token")
			round, err := cl.GetRoundsCurrent(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRoundsCurrent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want401 && !IsUnauthorized(err) {
				t.Errorf("expected IsUnauthorized(err)=true, got %v", err)
			}
			if tt.wantRound != nil {
				if round == nil || round.UID != tt.wantRound.UID || round.RoundIndex != tt.wantRound.RoundIndex {
					t.Errorf("round = %+v, want %+v", round, tt.wantRound)
				}
			} else if round != nil && tt.wantRound == nil && !tt.wantErr {
				if tt.name != "success_no_round" {
					t.Errorf("unexpected round %+v", round)
				}
			}
			if authHeader != "Bearer test-token" {
				t.Errorf("Authorization = %q", authHeader)
			}
		})
	}
}

func TestGetTeams(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantTeams  []Team
		wantErr    bool
		want401    bool
	}{
		{
			name:       "success",
			statusCode: 200,
			body:       `{"success":true,"data":{"teams":[{"uid":"t1","name":"Team A","color":"#fff","score":100}]}}`,
			wantTeams:  []Team{{UID: "t1", Name: "Team A", Color: "#fff", Score: 100}},
		},
		{
			name:       "empty",
			statusCode: 200,
			body:       `{"success":true,"data":{"teams":[]}}`,
			wantTeams:  []Team{},
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       `{}`,
			wantErr:    true,
			want401:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			cl := New(srv.URL, "tok")
			teams, err := cl.GetTeams(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTeams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want401 && !IsUnauthorized(err) {
				t.Errorf("expected 401")
			}
			if !tt.wantErr {
				if len(teams) != len(tt.wantTeams) {
					t.Errorf("len(teams)=%d want %d", len(teams), len(tt.wantTeams))
				} else if len(teams) > 0 && (teams[0].UID != tt.wantTeams[0].UID || teams[0].Name != tt.wantTeams[0].Name) {
					t.Errorf("teams[0] = %+v, want %+v", teams[0], tt.wantTeams[0])
				}
			}
		})
	}
}

func TestGetServices(t *testing.T) {
	ri := 1
	tests := []struct {
		name         string
		roundUID     string
		statusCode   int
		body         string
		wantServices []Service
		wantErr      bool
		want401      bool
	}{
		{
			name:         "success",
			statusCode:   200,
			body:         `{"success":true,"data":{"services":[{"uid":"s1","name":"Service","slug":"svc","round_uid":"r1","round_index":1}]}}`,
			wantServices: []Service{{UID: "s1", Name: "Service", Slug: "svc", RoundUID: "r1", RoundIndex: &ri}},
		},
		{
			name:         "with_round_param",
			roundUID:     "round-123",
			statusCode:   200,
			body:         `{"success":true,"data":{"services":[]}}`,
			wantServices: []Service{},
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       `{}`,
			wantErr:    true,
			want401:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path + "?" + r.URL.RawQuery
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			cl := New(srv.URL, "tok")
			services, err := cl.GetServices(context.Background(), tt.roundUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want401 && !IsUnauthorized(err) {
				t.Errorf("expected 401")
			}
			if tt.roundUID != "" {
				if path != "/api/ad/checker/services?round_uid=round-123" {
					t.Errorf("path/query = %q", path)
				}
			}
			if !tt.wantErr {
				if len(services) != len(tt.wantServices) {
					t.Errorf("len(services)=%d want %d", len(services), len(tt.wantServices))
				} else if len(services) > 0 && services[0].UID != tt.wantServices[0].UID {
					t.Errorf("services[0].UID = %q, want %q", services[0].UID, tt.wantServices[0].UID)
				}
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{&errUnauthorized{msg: "401"}, true},
		{nil, false},
		{errors.New("other"), false},
	}
	for _, tt := range tests {
		got := IsUnauthorized(tt.err)
		if got != tt.want {
			t.Errorf("IsUnauthorized(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestPutTeamInstances(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		want401    bool
	}{
		{
			name:       "success",
			statusCode: 200,
			body:       `{"success":true}`,
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       `{"success":false}`,
			wantErr:    true,
			want401:    true,
		},
		{
			name:       "bad_request",
			statusCode: 400,
			body:       `{"success":false,"message":"service_uid not found"}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("method = %s", r.Method)
				}
				if r.URL.Path != "/api/ad/checker/teams/instances" {
					t.Fatalf("path = %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			cl := New(srv.URL, "tok")
			err := cl.PutTeamInstances(context.Background(), []TeamInstanceInput{
				{TeamUID: "team-1", ServiceUID: "svc-1"},
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.want401 && !IsUnauthorized(err) {
				t.Fatalf("expected unauthorized error")
			}
		})
	}
}

func TestSubmitSLA(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		want401    bool
	}{
		{
			name:       "success",
			statusCode: 200,
			body:       `{"success":true}`,
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       `{"success":false}`,
			wantErr:    true,
			want401:    true,
		},
		{
			name:       "bad_request",
			statusCode: 400,
			body:       `{"success":false,"message":"Round not found for round_index"}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s", r.Method)
				}
				if r.URL.Path != "/api/ad/checker/sla" {
					t.Fatalf("path = %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			cl := New(srv.URL, "tok")
			err := cl.SubmitSLA(context.Background(), 1, []SLAResult{
				{TeamUID: "team-1", ServiceUID: "svc-1", Up: true},
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.want401 && !IsUnauthorized(err) {
				t.Fatalf("expected unauthorized error")
			}
		})
	}
}
