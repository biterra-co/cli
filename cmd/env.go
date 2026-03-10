package cmd

import (
	"fmt"
	"os"

	"github.com/geoctf/biterra-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	envFormat string
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print env vars for the checker process",
	Long:  "Outputs BITERRA_API_URL, BITERRA_CHECKER_TOKEN, BITERRA_TEAM_UID, BITERRA_SERVICE_UID. Use with eval $(biterra env) or --format dotenv for docker run --env-file.",
	RunE:  runEnv,
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringVar(&envFormat, "format", "shell", "Output format: shell (export FOO=bar) or dotenv")
}

func runEnv(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRequired()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config: run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
		}
		return err
	}
	switch envFormat {
	case "shell":
		fmt.Printf("export BITERRA_API_URL=%q\n", cfg.APIURL)
		fmt.Printf("export BITERRA_CHECKER_TOKEN=%q\n", cfg.CheckerToken)
		if cfg.TeamUID != "" {
			fmt.Printf("export BITERRA_TEAM_UID=%q\n", cfg.TeamUID)
		}
		if cfg.ServiceUID != "" {
			fmt.Printf("export BITERRA_SERVICE_UID=%q\n", cfg.ServiceUID)
		}
	case "dotenv":
		fmt.Printf("BITERRA_API_URL=%s\n", cfg.APIURL)
		fmt.Printf("BITERRA_CHECKER_TOKEN=%s\n", cfg.CheckerToken)
		if cfg.TeamUID != "" {
			fmt.Printf("BITERRA_TEAM_UID=%s\n", cfg.TeamUID)
		}
		if cfg.ServiceUID != "" {
			fmt.Printf("BITERRA_SERVICE_UID=%s\n", cfg.ServiceUID)
		}
	default:
		return fmt.Errorf("unknown format %q (use shell or dotenv)", envFormat)
	}
	return nil
}
