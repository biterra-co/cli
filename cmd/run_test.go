package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/biterra-co/cli/internal/client"
)

func TestResolveServiceRoundUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ad/checker/services" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"services":[{"uid":"svc-1","name":"Web","slug":"web","round_uid":"round-123","round_index":1}]}}`))
	}))
	defer srv.Close()

	cl := client.New(srv.URL, "tok")
	roundUID, err := resolveServiceRoundUID(context.Background(), cl, "svc-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if roundUID != "round-123" {
		t.Fatalf("roundUID=%s", roundUID)
	}
}

func TestResolveServiceRoundUID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"services":[]}}`))
	}))
	defer srv.Close()

	cl := client.New(srv.URL, "tok")
	_, err := resolveServiceRoundUID(context.Background(), cl, "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProbeHealth(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	downSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer downSrv.Close()

	httpClient := &http.Client{Timeout: 2 * time.Second}
	if !probeHealth(context.Background(), httpClient, okSrv.URL) {
		t.Fatalf("expected up=true for 200")
	}
	if probeHealth(context.Background(), httpClient, downSrv.URL) {
		t.Fatalf("expected up=false for 500")
	}
	if probeHealth(context.Background(), httpClient, "://bad-url") {
		t.Fatalf("expected up=false for invalid URL")
	}
}

func TestNormalizeProbeType(t *testing.T) {
	if got := normalizeProbeType("web"); got != "web" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType("BINARY"); got != "binary" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType(""); got != "none" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType("invalid"); got != "none" {
		t.Fatalf("got=%s", got)
	}
}

func TestValidateProbeConfig(t *testing.T) {
	if err := validateProbeConfig("none", "", ""); err != nil {
		t.Fatalf("none should be valid: %v", err)
	}
	if err := validateProbeConfig("web", "http://localhost/health", ""); err != nil {
		t.Fatalf("web should be valid: %v", err)
	}
	if err := validateProbeConfig("binary", "", "/tmp/flag"); err != nil {
		t.Fatalf("binary should be valid: %v", err)
	}
	if err := validateProbeConfig("web", "", ""); err == nil {
		t.Fatalf("expected error for missing web url")
	}
	if err := validateProbeConfig("binary", "", ""); err == nil {
		t.Fatalf("expected error for missing binary file")
	}
}

func TestEvaluateProbeBinary(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "flag-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("FLAG{ok}"); err != nil {
		t.Fatal(err)
	}
	if !evaluateProbe(context.Background(), &http.Client{Timeout: time.Second}, "binary", "", f.Name()) {
		t.Fatalf("expected binary probe up for non-empty file")
	}
	empty, err := os.CreateTemp(t.TempDir(), "flag-empty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()
	if evaluateProbe(context.Background(), &http.Client{Timeout: time.Second}, "binary", "", empty.Name()) {
		t.Fatalf("expected binary probe down for empty file")
	}
	if evaluateProbe(context.Background(), &http.Client{Timeout: time.Second}, "binary", "", "/no/such/file") {
		t.Fatalf("expected binary probe down for missing file")
	}
}
