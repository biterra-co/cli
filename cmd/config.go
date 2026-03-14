package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/biterra-co/cli/internal/config"
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
	configSetCmd.Flags().StringVar(&configSetProbeType, "probe-type", "", "Probe type: web, binary, or none")
	configSetCmd.Flags().StringVar(&configSetProbeWebURL, "probe-web-url", "", "Probe URL for web checks (used when probe-type=web)")
	configSetCmd.Flags().StringVar(&configSetProbeBinaryFile, "probe-binary-flag-file", "", "Flag file path for binary checks (used when probe-type=binary)")
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	cfg, path, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config found — run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
		}
		return err
	}
	if path != "" {
		fmt.Printf("Config file: %s\n", path)
	} else {
		fmt.Println("Config file: (env only)")
	}
	fmt.Printf("api_url: %s\n", cfg.APIURL)
	if configShowToken {
		fmt.Printf("checker_token: %s\n", cfg.CheckerToken)
	} else {
		if cfg.CheckerToken != "" {
			fmt.Println("checker_token: ***")
		} else {
			fmt.Println("checker_token: (not set)")
		}
	}
	if cfg.CustomerPortalURL != "" {
		fmt.Printf("customer_portal_url: %s\n", cfg.CustomerPortalURL)
	}
	fmt.Printf("team_uid: %s\n", cfg.TeamUID)
	fmt.Printf("service_uid: %s\n", cfg.ServiceUID)
	fmt.Printf("probe_type: %s\n", cfg.ProbeType)
	fmt.Printf("probe_web_url: %s\n", cfg.ProbeWebURL)
	fmt.Printf("probe_binary_flag_file: %s\n", cfg.ProbeBinaryFile)
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
	if cfg.APIURL == "" || cfg.CheckerToken == "" {
		return fmt.Errorf("api_url and token are required — use --api-url and --token")
	}
	switch strings.ToLower(strings.TrimSpace(cfg.ProbeType)) {
	case "", "none", "web", "binary":
	default:
		return fmt.Errorf("invalid probe_type %q (use web, binary, or none)", cfg.ProbeType)
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "web" && strings.TrimSpace(cfg.ProbeWebURL) == "" {
		return fmt.Errorf("probe_type=web requires --probe-web-url (or set probe_web_url in config)")
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProbeType)) == "binary" && strings.TrimSpace(cfg.ProbeBinaryFile) == "" {
		return fmt.Errorf("probe_type=binary requires --probe-binary-flag-file (or set probe_binary_flag_file in config)")
	}
	return config.Save(cfg)
}
