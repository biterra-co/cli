package cmd

import (
	"fmt"
	"os"

	"github.com/biterra-co/cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	configShowToken bool
	configSetAPIURL string
	configSetToken  string
	configSetTeam   string
	configSetSvc    string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Get or set config",
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Print current config (token masked unless --show-token)",
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set config values (non-interactive)",
	RunE:  runConfigSet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd, configSetCmd)

	configGetCmd.Flags().BoolVar(&configShowToken, "show-token", false, "Print token (for scripting)")

	configSetCmd.Flags().StringVar(&configSetAPIURL, "api-url", "", "API base URL")
	configSetCmd.Flags().StringVar(&configSetToken, "token", "", "Checker token")
	configSetCmd.Flags().StringVar(&configSetTeam, "team-uid", "", "Team UID (optional)")
	configSetCmd.Flags().StringVar(&configSetSvc, "service-uid", "", "Service UID (optional)")
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	cfg, path, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config found; run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
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
	fmt.Printf("team_uid: %s\n", cfg.TeamUID)
	fmt.Printf("service_uid: %s\n", cfg.ServiceUID)
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
	if configSetTeam != "" {
		cfg.TeamUID = configSetTeam
	}
	if configSetSvc != "" {
		cfg.ServiceUID = configSetSvc
	}
	if cfg.APIURL == "" || cfg.CheckerToken == "" {
		return fmt.Errorf("api_url and token are required (use --api-url and --token)")
	}
	return config.Save(cfg)
}
