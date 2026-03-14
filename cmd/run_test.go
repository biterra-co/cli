package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
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
