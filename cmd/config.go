package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/biterra-co/cli/internal/config"
	"github.com/biterra-co/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	configShowToken            bool
	configSetAPIURL            string
	configSetToken             string
	configSetCustomerPortalURL string
	configSetTeam              string
	configSetSvc               string
	configSetProbeType         string
	configSetProbeWebURL       string
	configSetProbeBinaryFile   string
	configSetProbeTCPAddress   string
	configSetProbeCommand      string
	configSetProbeGRPCAddress  string
	configSetProbeGRPCService  string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Get or set config",
}

var configGetCmd = &cobra.Command{
	Use:          "get",
	Short:        "Print current config (token masked unless --show-token)",
	RunE:         runConfigGet,
	SilenceUsage: true,
}

var configSetCmd = &cobra.Command{
	Use:          "set",
	Short:        "Set config values (non-interactive)",
	RunE:         runConfigSet,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd, configSetCmd)

	configGetCmd.Flags().BoolVar(&configShowToken, "show-token", false, "Print token (for scripting)")

	configSetCmd.Flags().StringVar(&configSetAPIURL, "api-url", "", "API base URL")
	configSetCmd.Flags().StringVar(&configSetToken, "token", "", "Checker token")
	configSetCmd.Flags().StringVar(&configSetCustomerPortalURL, "customer-portal-url", "", "Customer portal URL for token setup (optional, default https://ctf.biterra.co)")
	configSetCmd.Flags().StringVar(&configSetTeam, "team-uid", "", "Team UID (optional)")
	configSetCmd.Flags().StringVar(&configSetSvc, "service-uid", "", "Service UID (optional)")
	configSetCmd.Flags().StringVar(&configSetProbeType, "probe-type", "", "Probe type: web, binary, tcp, command, or grpc")
	configSetCmd.Flags().StringVar(&configSetProbeWebURL, "probe-web-url", "", "Probe URL for web checks (used when probe-type=web)")
	configSetCmd.Flags().StringVar(&configSetProbeBinaryFile, "probe-binary-flag-file", "", "Flag file path for binary checks (used when probe-type=binary)")
	configSetCmd.Flags().StringVar(&configSetProbeTCPAddress, "probe-tcp-address", "", "host:port for TCP checks (used when probe-type=tcp)")
	configSetCmd.Flags().StringVar(&configSetProbeCommand, "probe-command", "", "Local command to run; exit 0 means up (used when probe-type=command)")
	configSetCmd.Flags().StringVar(&configSetProbeGRPCAddress, "probe-grpc-address", "", "host:port for gRPC health checks (used when probe-type=grpc)")
	configSetCmd.Flags().StringVar(&configSetProbeGRPCService, "probe-grpc-service", "", "Optional gRPC health service name (used when probe-type=grpc)")
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	cfg, path, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config found — run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
		}
		return err
	}
	ui.Bold("Current configuration")
	ui.Muted("Values loaded from config files and environment overrides.")
	ui.Rule()
	if path != "" {
		ui.KeyValue("Config file", path)
	} else {
		ui.KeyValue("Config file", "(env only)")
	}
	ui.Blank()

	ui.Section("Connection")
	ui.KeyValue("api_url", displayValue(cfg.APIURL))
	ui.KeyValue("checker_token", displayToken(cfg.CheckerToken, configShowToken))
	ui.KeyValue("customer_portal_url", displayValue(cfg.CustomerPortalURL))

	ui.Section("Checker")
	ui.KeyValue("team_uid", displayValue(cfg.TeamUID))
	ui.KeyValue("service_uid", displayValue(cfg.ServiceUID))

	ui.Section("Probe")
	ui.KeyValue("probe_type", displayValue(cfg.ProbeType))
	ui.KeyValue("probe_web_url", displayValue(cfg.ProbeWebURL))
	ui.KeyValue("probe_binary_flag_file", displayValue(cfg.ProbeBinaryFile))
	ui.KeyValue("probe_tcp_address", displayValue(cfg.ProbeTCPAddress))
	ui.KeyValue("probe_command", displayValue(cfg.ProbeCommand))
	ui.KeyValue("probe_grpc_address", displayValue(cfg.ProbeGRPCAddress))
	ui.KeyValue("probe_grpc_service", displayValue(cfg.ProbeGRPCService))
	ui.Rule()
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.Load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// Apply env override then flags
	if v := os.Getenv("BITERRA_API_URL"); v != "" {
		cfg.APIURL = v
	}
	if v := os.Getenv("BITERRA_CHECKER_TOKEN"); v != "" {
		cfg.CheckerToken = v
	}
	if configSetAPIURL != "" {
		cfg.APIURL = configSetAPIURL
	}
	if configSetToken != "" {
		cfg.CheckerToken = configSetToken
	}
	if configSetCustomerPortalURL != "" {
		cfg.CustomerPortalURL = configSetCustomerPortalURL
	}
	if configSetTeam != "" {
		cfg.TeamUID = configSetTeam
	}
	if configSetSvc != "" {
		cfg.ServiceUID = configSetSvc
	}
	if configSetProbeType != "" {
		cfg.ProbeType = strings.ToLower(strings.TrimSpace(configSetProbeType))
	}
	if configSetProbeWebURL != "" {
		cfg.ProbeWebURL = strings.TrimSpace(configSetProbeWebURL)
	}
	if configSetProbeBinaryFile != "" {
		cfg.ProbeBinaryFile = strings.TrimSpace(configSetProbeBinaryFile)
	}
	if configSetProbeTCPAddress != "" {
		cfg.ProbeTCPAddress = strings.TrimSpace(configSetProbeTCPAddress)
	}
	if configSetProbeCommand != "" {
		cfg.ProbeCommand = strings.TrimSpace(configSetProbeCommand)
	}
	if configSetProbeGRPCAddress != "" {
		cfg.ProbeGRPCAddress = strings.TrimSpace(configSetProbeGRPCAddress)
	}
	if configSetProbeGRPCService != "" {
		cfg.ProbeGRPCService = strings.TrimSpace(configSetProbeGRPCService)
	}
	if cfg.APIURL == "" || cfg.CheckerToken == "" {
		return fmt.Errorf("api_url and token are required — use --api-url and --token")
	}
	switch strings.ToLower(strings.TrimSpace(cfg.ProbeType)) {
	case "", "web", "binary", "tcp", "command", "grpc":
	default:
		return fmt.Errorf("invalid probe_type %q (use web, binary, tcp, command, or grpc)", cfg.ProbeType)
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "web" && strings.TrimSpace(cfg.ProbeWebURL) == "" {
		return fmt.Errorf("probe_type=web requires --probe-web-url (or set probe_web_url in config)")
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "binary" && strings.TrimSpace(cfg.ProbeBinaryFile) == "" {
		return fmt.Errorf("probe_type=binary requires --probe-binary-flag-file (or set probe_binary_flag_file in config)")
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "tcp" && strings.TrimSpace(cfg.ProbeTCPAddress) == "" {
		return fmt.Errorf("probe_type=tcp requires --probe-tcp-address (or set probe_tcp_address in config)")
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "command" && strings.TrimSpace(cfg.ProbeCommand) == "" {
		return fmt.Errorf("probe_type=command requires --probe-command (or set probe_command in config)")
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "grpc" && strings.TrimSpace(cfg.ProbeGRPCAddress) == "" {
		return fmt.Errorf("probe_type=grpc requires --probe-grpc-address (or set probe_grpc_address in config)")
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	lines := []string{
		fmt.Sprintf("API URL: %s", cfg.APIURL),
		fmt.Sprintf("Team UID: %s", displayValue(cfg.TeamUID)),
		fmt.Sprintf("Service UID: %s", displayValue(cfg.ServiceUID)),
		fmt.Sprintf("Probe Type: %s", displayValue(cfg.ProbeType)),
	}
	ui.SuccessBlock("Config saved.", lines)
	return nil
}

func displayValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(not set)"
	}
	return v
}

func displayToken(token string, show bool) string {
	if strings.TrimSpace(token) == "" {
		return "(not set)"
	}
	if show {
		return token
	}
	return "***"
}
