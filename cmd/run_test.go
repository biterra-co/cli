package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/biterra-co/cli/internal/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
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

func TestCheckInTeamInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ad/checker/teams/instances" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"instances":[]}}`))
	}))
	defer srv.Close()

	cl := client.New(srv.URL, "tok")
	if err := checkInTeamInstance(context.Background(), cl, "team-1", "svc-1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
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
	if got := normalizeProbeType("tcp"); got != "tcp" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType("command"); got != "command" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType("grpc"); got != "grpc" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType(""); got != "" {
		t.Fatalf("got=%s", got)
	}
	if got := normalizeProbeType("invalid"); got != "" {
		t.Fatalf("got=%s", got)
	}
}

func TestValidateProbeConfig(t *testing.T) {
	if err := validateProbeConfig(probeConfig{Type: "web", WebURL: "http://localhost/health"}); err != nil {
		t.Fatalf("web should be valid: %v", err)
	}
	if err := validateProbeConfig(probeConfig{Type: "binary", BinaryFile: "/tmp/flag"}); err != nil {
		t.Fatalf("binary should be valid: %v", err)
	}
	if err := validateProbeConfig(probeConfig{Type: "tcp", TCPAddress: "127.0.0.1:31337"}); err != nil {
		t.Fatalf("tcp should be valid: %v", err)
	}
	if err := validateProbeConfig(probeConfig{Type: "command", Command: "echo ok"}); err != nil {
		t.Fatalf("command should be valid: %v", err)
	}
	if err := validateProbeConfig(probeConfig{Type: "grpc", GRPCAddress: "127.0.0.1:50051"}); err != nil {
		t.Fatalf("grpc should be valid: %v", err)
	}
	if err := validateProbeConfig(probeConfig{Type: "web"}); err == nil {
		t.Fatalf("expected error for missing web url")
	}
	if err := validateProbeConfig(probeConfig{Type: "binary"}); err == nil {
		t.Fatalf("expected error for missing binary file")
	}
	if err := validateProbeConfig(probeConfig{Type: "tcp"}); err == nil {
		t.Fatalf("expected error for missing tcp address")
	}
	if err := validateProbeConfig(probeConfig{Type: "command"}); err == nil {
		t.Fatalf("expected error for missing command")
	}
	if err := validateProbeConfig(probeConfig{Type: "grpc"}); err == nil {
		t.Fatalf("expected error for missing grpc address")
	}
	if err := validateProbeConfig(probeConfig{}); err == nil {
		t.Fatalf("expected error for missing probe type")
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
	if !evaluateProbe(context.Background(), probeConfig{Type: "binary", BinaryFile: f.Name()}) {
		t.Fatalf("expected binary probe up for non-empty file")
	}
	empty, err := os.CreateTemp(t.TempDir(), "flag-empty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()
	if evaluateProbe(context.Background(), probeConfig{Type: "binary", BinaryFile: empty.Name()}) {
		t.Fatalf("expected binary probe down for empty file")
	}
	if evaluateProbe(context.Background(), probeConfig{Type: "binary", BinaryFile: "/no/such/file"}) {
		t.Fatalf("expected binary probe down for missing file")
	}
}

func TestEvaluateProbeTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	if !evaluateProbe(context.Background(), probeConfig{Type: "tcp", TCPAddress: ln.Addr().String(), Timeout: time.Second}) {
		t.Fatalf("expected tcp probe up for listening socket")
	}

	closedAddress := ln.Addr().String()
	_ = ln.Close()
	if evaluateProbe(context.Background(), probeConfig{Type: "tcp", TCPAddress: closedAddress, Timeout: 200 * time.Millisecond}) {
		t.Fatalf("expected tcp probe down for closed socket")
	}
}

func TestEvaluateProbeCommand(t *testing.T) {
	okCmd := helperCommand(t, "exit-zero")
	if !evaluateProbe(context.Background(), probeConfig{Type: "command", Command: okCmd, Timeout: time.Second}) {
		t.Fatalf("expected command probe up for exit 0")
	}

	failCmd := helperCommand(t, "exit-one")
	if evaluateProbe(context.Background(), probeConfig{Type: "command", Command: failCmd, Timeout: time.Second}) {
		t.Fatalf("expected command probe down for non-zero exit")
	}
}

func TestEvaluateProbeGRPC(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("demo.Service", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	go func() {
		_ = server.Serve(lis)
	}()
	defer server.Stop()

	cfg := probeConfig{
		Type:        "grpc",
		GRPCAddress: lis.Addr().String(),
		GRPCService: "demo.Service",
		Timeout:     time.Second,
	}
	if !evaluateProbe(context.Background(), cfg) {
		t.Fatalf("expected grpc probe up for serving health endpoint")
	}

	cfg.GRPCService = "missing.Service"
	if evaluateProbe(context.Background(), cfg) {
		t.Fatalf("expected grpc probe down for unknown service")
	}
}

func helperCommand(t *testing.T, mode string) string {
	t.Helper()
	return fmt.Sprintf("GO_WANT_HELPER_PROCESS=1 %q -test.run=TestProbeCommandHelper -- %s", os.Args[0], mode)
}

func TestProbeCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" && i+1 < len(args) {
			switch args[i+1] {
			case "exit-zero":
				os.Exit(0)
			case "exit-one":
				os.Exit(1)
			}
		}
	}
	os.Exit(2)
}
