package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/biterra-co/cli/internal/client"
	"github.com/biterra-co/cli/internal/config"
	"github.com/biterra-co/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	runIntervalSeconds int
	runHealthURL       string
	runHTTPTimeoutSec  int
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
	runCmd.Flags().IntVar(&runIntervalSeconds, "interval-seconds", 30, "SLA loop interval in seconds")
	runCmd.Flags().StringVar(&runHealthURL, "health-url", "", "Optional local health URL to probe (2xx = up)")
	runCmd.Flags().IntVar(&runHTTPTimeoutSec, "http-timeout-seconds", 10, "HTTP timeout for optional health URL probe")
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
	if runIntervalSeconds < 1 {
		return fmt.Errorf("--interval-seconds must be >= 1")
	}
	if runHTTPTimeoutSec < 1 {
		return fmt.Errorf("--http-timeout-seconds must be >= 1")
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cl := client.New(cfg.APIURL, cfg.CheckerToken)

	// Resolve selected service once so we know which round this checker belongs to.
	ui.StepStart("Resolving selected service... ")
	serviceRoundUID, err := resolveServiceRoundUID(ctx, cl, cfg.ServiceUID)
	if err != nil {
		ui.StepFail()
		return err
	}
	ui.StepOK(serviceRoundUID)
	ui.Blank()

	var healthClient *http.Client
	healthURL := strings.TrimSpace(runHealthURL)
	if healthURL != "" {
		healthClient = &http.Client{Timeout: time.Duration(runHTTPTimeoutSec) * time.Second}
		ui.KeyValue("Health URL", healthURL)
	} else {
		ui.Muted("No --health-url provided; defaulting SLA status to up=true.")
	}
	ui.KeyValue("Team UID", cfg.TeamUID)
	ui.KeyValue("Service UID", cfg.ServiceUID)
	ui.KeyValue("Service Round UID", serviceRoundUID)
	ui.KeyValue("Interval", fmt.Sprintf("%ds", runIntervalSeconds))
	ui.Blank()

	ticker := time.NewTicker(time.Duration(runIntervalSeconds) * time.Second)
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

			up := true
			if healthClient != nil {
				up = probeHealth(ctx, healthClient, healthURL)
			}

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
