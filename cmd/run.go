package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/biterra-co/cli/internal/client"
	"github.com/biterra-co/cli/internal/config"
	"github.com/biterra-co/cli/internal/ui"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	runHealthURL       string
	runProbeTimeoutSec int
)

var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Run checker SLA loop",
	Long:         "Submits SLA only when the current round matches the selected service round.",
	RunE:         runRun,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&runHealthURL, "health-url", "", "Optional web health URL override (2xx = up)")
	runCmd.Flags().IntVar(&runProbeTimeoutSec, "probe-timeout-seconds", 10, "Timeout in seconds for probe execution")
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRequired()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config found — run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
		}
		return err
	}
	if strings.TrimSpace(cfg.TeamUID) == "" || strings.TrimSpace(cfg.ServiceUID) == "" {
		return fmt.Errorf("team_uid and service_uid are required — run 'biterra init' or set via 'biterra config set --team-uid ... --service-uid ...'")
	}
	if runProbeTimeoutSec < 1 {
		return fmt.Errorf("--probe-timeout-seconds must be >= 1")
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cl := client.New(cfg.APIURL, cfg.CheckerToken)
	ui.Bold("Checker run")
	ui.Muted("Starting the SLA loop for the configured team and service.")
	ui.Rule()
	ui.Blank()

	// Resolve selected service once so we know which round this checker belongs to.
	ui.StepStart("Resolving selected service... ")
	serviceRoundUID, err := resolveServiceRoundUID(ctx, cl, cfg.ServiceUID)
	if err != nil {
		ui.StepFail()
		return err
	}
	ui.StepOK(serviceRoundUID)
	ui.Blank()

	ui.StepStart("Checking in checker instance... ")
	if err := checkInTeamInstance(ctx, cl, cfg.TeamUID, cfg.ServiceUID); err != nil {
		ui.StepFail()
		if client.IsUnauthorized(err) {
			return fmt.Errorf("token invalid or expired during checker check-in")
		}
		return fmt.Errorf("checker check-in failed: %w", err)
	}
	ui.StepOK("registered")
	ui.Blank()

	ui.StepStart("Loading checker settings... ")
	runtimeSettings, err := cl.GetRuntimeSettings(ctx)
	if err != nil {
		ui.StepFail()
		if client.IsUnauthorized(err) {
			return fmt.Errorf("token invalid or expired while loading checker settings")
		}
		return fmt.Errorf("could not load checker settings: %w", err)
	}
	ui.StepOK(fmt.Sprintf("%ds", runtimeSettings.TickIntervalSeconds))
	ui.Blank()

	probeType := normalizeProbeType(cfg.ProbeType)
	healthURL := strings.TrimSpace(runHealthURL)
	if healthURL == "" {
		healthURL = strings.TrimSpace(cfg.ProbeWebURL)
	}
	probeCfg := probeConfig{
		Type:        probeType,
		WebURL:      healthURL,
		BinaryFile:  strings.TrimSpace(cfg.ProbeBinaryFile),
		TCPAddress:  strings.TrimSpace(cfg.ProbeTCPAddress),
		Command:     strings.TrimSpace(cfg.ProbeCommand),
		GRPCAddress: strings.TrimSpace(cfg.ProbeGRPCAddress),
		GRPCService: strings.TrimSpace(cfg.ProbeGRPCService),
		Timeout:     time.Duration(runProbeTimeoutSec) * time.Second,
		HTTPClient:  &http.Client{Timeout: time.Duration(runProbeTimeoutSec) * time.Second},
	}
	if err := validateProbeConfig(probeCfg); err != nil {
		return err
	}

	ui.KeyValue("Probe Type", probeType)
	printProbeConfig(probeCfg)
	ui.KeyValue("Team UID", cfg.TeamUID)
	ui.KeyValue("Service UID", cfg.ServiceUID)
	ui.KeyValue("Service Round UID", serviceRoundUID)
	ui.KeyValue("Tick Interval", fmt.Sprintf("%ds", runtimeSettings.TickIntervalSeconds))
	ui.KeyValue("Probe Timeout", fmt.Sprintf("%ds", runProbeTimeoutSec))
	ui.Rule()
	ui.Blank()

	ticker := time.NewTicker(time.Duration(runtimeSettings.TickIntervalSeconds) * time.Second)
	defer ticker.Stop()

	ui.Info("Checker loop started. Press Ctrl+C to stop.")

	for {
		select {
		case <-ctx.Done():
			ui.Info("Shutdown signal received. Exiting.")
			return nil
		case <-ticker.C:
			round, err := cl.GetRoundsCurrent(ctx)
			if err != nil {
				if client.IsUnauthorized(err) {
					return fmt.Errorf("token invalid or expired during run")
				}
				ui.Warning("Could not fetch current round: %v", err)
				continue
			}
			if round == nil {
				ui.Muted("Waiting for active round...")
				continue
			}
			if round.UID != serviceRoundUID {
				ui.Muted("Round %d active but not this service round; waiting...", round.RoundIndex)
				continue
			}

			up := evaluateProbe(ctx, probeCfg)

			err = cl.SubmitSLA(ctx, round.RoundIndex, []client.SLAResult{
				{
					TeamUID:    cfg.TeamUID,
					ServiceUID: cfg.ServiceUID,
					Up:         up,
				},
			})
			if err != nil {
				if client.IsUnauthorized(err) {
					return fmt.Errorf("token invalid or expired during SLA submit")
				}
				ui.Warning("SLA submit failed (round %d): %v", round.RoundIndex, err)
				continue
			}
			if up {
				ui.Success("SLA submitted (round %d): up", round.RoundIndex)
			} else {
				ui.Warning("SLA submitted (round %d): down", round.RoundIndex)
			}
		}
	}
}

func checkInTeamInstance(ctx context.Context, cl *client.Client, teamUID, serviceUID string) error {
	return cl.PutTeamInstances(ctx, []client.TeamInstanceInput{
		{
			TeamUID:    teamUID,
			ServiceUID: serviceUID,
		},
	})
}

func resolveServiceRoundUID(ctx context.Context, cl *client.Client, serviceUID string) (string, error) {
	services, err := cl.GetServices(ctx, "")
	if err != nil {
		if client.IsUnauthorized(err) {
			return "", fmt.Errorf("token invalid or expired")
		}
		return "", fmt.Errorf("could not load services: %w", err)
	}
	for _, s := range services {
		if strings.TrimSpace(s.UID) == strings.TrimSpace(serviceUID) {
			if strings.TrimSpace(s.RoundUID) == "" {
				return "", fmt.Errorf("service %s has no round_uid", serviceUID)
			}
			return s.RoundUID, nil
		}
	}
	return "", fmt.Errorf("service_uid %s not found in checker services list", serviceUID)
}

func probeHealth(ctx context.Context, httpClient *http.Client, healthURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func normalizeProbeType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "web", "binary", "tcp", "command", "grpc":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}

type probeConfig struct {
	Type        string
	WebURL      string
	BinaryFile  string
	TCPAddress  string
	Command     string
	GRPCAddress string
	GRPCService string
	Timeout     time.Duration
	HTTPClient  *http.Client
}

func validateProbeConfig(cfg probeConfig) error {
	switch cfg.Type {
	case "web":
		if strings.TrimSpace(cfg.WebURL) == "" {
			return fmt.Errorf("probe_type=web requires probe_web_url in config or --health-url override")
		}
	case "binary":
		if strings.TrimSpace(cfg.BinaryFile) == "" {
			return fmt.Errorf("probe_type=binary requires probe_binary_flag_file in config")
		}
	case "tcp":
		if strings.TrimSpace(cfg.TCPAddress) == "" {
			return fmt.Errorf("probe_type=tcp requires probe_tcp_address in config")
		}
	case "command":
		if strings.TrimSpace(cfg.Command) == "" {
			return fmt.Errorf("probe_type=command requires probe_command in config")
		}
	case "grpc":
		if strings.TrimSpace(cfg.GRPCAddress) == "" {
			return fmt.Errorf("probe_type=grpc requires probe_grpc_address in config")
		}
	default:
		return fmt.Errorf("probe_type must be web, binary, tcp, command, or grpc")
	}
	return nil
}

func printProbeConfig(cfg probeConfig) {
	switch cfg.Type {
	case "web":
		ui.KeyValue("Health URL", cfg.WebURL)
	case "binary":
		ui.KeyValue("Flag file", cfg.BinaryFile)
	case "tcp":
		ui.KeyValue("TCP address", cfg.TCPAddress)
	case "command":
		ui.KeyValue("Probe command", cfg.Command)
	case "grpc":
		ui.KeyValue("gRPC address", cfg.GRPCAddress)
		if cfg.GRPCService != "" {
			ui.KeyValue("gRPC service", cfg.GRPCService)
		}
	}
}

func evaluateProbe(ctx context.Context, cfg probeConfig) bool {
	switch cfg.Type {
	case "web":
		return probeHealth(ctx, cfg.HTTPClient, cfg.WebURL)
	case "binary":
		b, err := os.ReadFile(cfg.BinaryFile)
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(b)) != ""
	case "tcp":
		return probeTCP(ctx, cfg.Timeout, cfg.TCPAddress)
	case "command":
		return probeCommand(ctx, cfg.Timeout, cfg.Command)
	case "grpc":
		return probeGRPC(ctx, cfg.Timeout, cfg.GRPCAddress, cfg.GRPCService)
	default:
		return false
	}
}

func probeTCP(ctx context.Context, timeout time.Duration, address string) bool {
	dialer := &net.Dialer{}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dialer.DialContext(probeCtx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func probeCommand(ctx context.Context, timeout time.Duration, command string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, args := shellCommand(command)
	cmd := exec.CommandContext(probeCtx, name, args...)
	return cmd.Run() == nil
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-c", command}
}

func probeGRPC(ctx context.Context, timeout time.Duration, address, service string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false
	}
	defer conn.Close()

	resp, err := grpc_health_v1.NewHealthClient(conn).Check(probeCtx, &grpc_health_v1.HealthCheckRequest{
		Service: service,
	})
	if err != nil {
		return false
	}
	return resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING
}
