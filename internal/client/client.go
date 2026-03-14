package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const checkerPath = "/api/ad/checker"

// Client calls the Checker API with Bearer token.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New builds a client. baseURL should not have a trailing slash.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		http:    &http.Client{},
	}
}

func (c *Client) url(path string) string {
	return c.baseURL + checkerPath + path
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	var err error
	var req *http.Request
	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.url(path), reqBody)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.url(path), nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

// API response wrapper
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

// Round is the current round payload.
type Round struct {
	UID        string  `json:"uid"`
	RoundIndex int     `json:"round_index"`
	StartedAt  string  `json:"started_at"`
	EndedAt    *string `json:"ended_at,omitempty"`
}

type RuntimeSettings struct {
	Round               *Round `json:"round"`
	TickIntervalSeconds int    `json:"tick_interval_seconds"`
}

const defaultTickIntervalSeconds = 30

// GetRuntimeSettings returns the current round plus checker runtime settings.
func (c *Client) GetRuntimeSettings(ctx context.Context) (*RuntimeSettings, error) {
	resp, err := c.do(ctx, http.MethodGet, "/rounds/current", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &errUnauthorized{msg: "401 Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rounds/current: %s", resp.Status)
	}
	var out apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	data := &RuntimeSettings{TickIntervalSeconds: defaultTickIntervalSeconds}
	if len(out.Data) > 0 {
		if err := json.Unmarshal(out.Data, data); err != nil {
			return nil, err
		}
	}
	if data.TickIntervalSeconds < 1 {
		data.TickIntervalSeconds = defaultTickIntervalSeconds
	}
	return data, nil
}

// GetRoundsCurrent returns the current round or nil if none. Returns error on 401 or other failure.
func (c *Client) GetRoundsCurrent(ctx context.Context) (*Round, error) {
	settings, err := c.GetRuntimeSettings(ctx)
	if err != nil {
		return nil, err
	}
	return settings.Round, nil
}

type errUnauthorized struct{ msg string }

func (e *errUnauthorized) Error() string { return e.msg }

// IsUnauthorized reports whether err is a 401 from the API.
func IsUnauthorized(err error) bool {
	_, ok := err.(*errUnauthorized)
	return ok
}

// Team is a team row from GET /teams.
type Team struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Score int    `json:"score"`
}

// GetTeams returns teams for the world.
func (c *Client) GetTeams(ctx context.Context) ([]Team, error) {
	resp, err := c.do(ctx, http.MethodGet, "/teams", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &errUnauthorized{msg: "401 Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("teams: %s", resp.Status)
	}
	var out apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	var data struct {
		Teams []Team `json:"teams"`
	}
	if len(out.Data) > 0 {
		if err := json.Unmarshal(out.Data, &data); err != nil {
			return nil, err
		}
	}
	return data.Teams, nil
}

// Service is a checker service row from GET /services.
type Service struct {
	UID        string `json:"uid"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	RoundUID   string `json:"round_uid"`
	RoundIndex *int   `json:"round_index,omitempty"`
}

// GetServices returns services. roundUID can be empty for current round.
func (c *Client) GetServices(ctx context.Context, roundUID string) ([]Service, error) {
	path := "/services"
	if roundUID != "" {
		path += "?round_uid=" + url.QueryEscape(roundUID)
	}
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &errUnauthorized{msg: "401 Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("services: %s", resp.Status)
	}
	var out apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	var data struct {
		Services []Service `json:"services"`
	}
	if len(out.Data) > 0 {
		if err := json.Unmarshal(out.Data, &data); err != nil {
			return nil, err
		}
	}
	return data.Services, nil
}

// TeamInstanceInput is one team/service row for checker check-in.
// instance_url is optional; checker can send only team_uid+service_uid.
type TeamInstanceInput struct {
	TeamUID     string  `json:"team_uid"`
	ServiceUID  string  `json:"service_uid"`
	InstanceURL *string `json:"instance_url,omitempty"`
}

// PutTeamInstances performs checker check-in/upsert via PUT /teams/instances.
func (c *Client) PutTeamInstances(ctx context.Context, instances []TeamInstanceInput) error {
	body := map[string]interface{}{
		"instances": instances,
	}
	resp, err := c.do(ctx, http.MethodPut, "/teams/instances", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return &errUnauthorized{msg: "401 Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		msg := readErrorMessage(resp)
		if msg != "" {
			return fmt.Errorf("teams/instances: %s (%s)", resp.Status, msg)
		}
		return fmt.Errorf("teams/instances: %s", resp.Status)
	}
	return nil
}

// SLAResult is one SLA result for a team/service.
type SLAResult struct {
	TeamUID    string `json:"team_uid"`
	ServiceUID string `json:"service_uid"`
	Up         bool   `json:"up"`
}

// SubmitSLA posts SLA status for one round.
func (c *Client) SubmitSLA(ctx context.Context, roundIndex int, results []SLAResult) error {
	body := map[string]interface{}{
		"round_index": roundIndex,
		"results":     results,
	}
	resp, err := c.do(ctx, http.MethodPost, "/sla", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return &errUnauthorized{msg: "401 Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		msg := readErrorMessage(resp)
		if msg != "" {
			return fmt.Errorf("sla: %s (%s)", resp.Status, msg)
		}
		return fmt.Errorf("sla: %s", resp.Status)
	}
	return nil
}

func readErrorMessage(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return ""
	}
	var out struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(b, &out); err == nil && strings.TrimSpace(out.Message) != "" {
		return out.Message
	}
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
