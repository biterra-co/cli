package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// TokenInfo calls the customer portal POST /api/account/developer/token-info with body { token }.
// with the raw token and returns the world API base URL for that token.
func TokenInfo(ctx context.Context, customerPortalURL, rawToken string) (apiBaseURL string, err error) {
	base := strings.TrimSuffix(customerPortalURL, "/")
	apiURL := base + "/api/account/developer/token-info"
	body := map[string]string{"token": rawToken}
	encoded, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach the customer portal: %w", err)
	}
	defer resp.Body.Close()

	// Read body once so we can detect HTML and still decode JSON
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response: %w", err)
	}

	// If we got HTML (e.g. 404 page, login redirect), give a clear message
	trimmed := bytes.TrimLeft(raw, " \t\n\r")
	if len(trimmed) > 0 && trimmed[0] == '<' {
		return "", &ErrPortalNotReady{
			URL:    base,
			Status: resp.StatusCode,
			Hint:   "The portal returned a web page instead of data. Make sure the customer portal is running at this URL and the Developer token-info API is available.",
		}
	}

	var out struct {
		Success    bool   `json:"success"`
		Message    string `json:"message"`
		APIBaseURL string `json:"api_base_url"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		if resp.StatusCode != http.StatusOK {
			return "", &ErrPortalNotReady{
				URL:    base,
				Status: resp.StatusCode,
				Hint:   "Check that the customer portal is running and the URL is correct.",
			}
		}
		return "", fmt.Errorf("invalid response from portal: %w", err)
	}

	if !out.Success {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("invalid or expired token — create a new token in the Developer section and try again")
		}
		msg := out.Message
		if msg == "" {
			msg = "request was not successful"
		}
		return "", fmt.Errorf("%s", msg)
	}
	// Empty api_base_url is OK: world may have no domain; caller can prompt for API URL
	return strings.TrimSuffix(out.APIBaseURL, "/"), nil
}

// ErrPortalNotReady is returned when the customer portal returns non-JSON (e.g. HTML).
type ErrPortalNotReady struct {
	URL    string
	Status int
	Hint   string
}

func (e *ErrPortalNotReady) Error() string {
	return fmt.Sprintf("customer portal at %s returned an unexpected response (HTTP %d). %s", e.URL, e.Status, e.Hint)
}
