package cmd

import (
	"fmt"
	"os"

	"github.com/biterra-co/cli/internal/client"
	"github.com/biterra-co/cli/internal/config"
	"github.com/biterra-co/cli/internal/ui"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:          "check",
	Short:        "Verify your config and token against the world API",
	Long:         "Calls the checker API to confirm your token is valid and prints the current round. Exits 0 on success.",
	RunE:         runCheck,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRequired()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config found — run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
		}
		return err
	}
	cl := client.New(cfg.APIURL, cfg.CheckerToken)
	ui.Bold("Checker status")
	ui.Muted("Verifying your token and runtime settings against the world API.")
	ui.Rule()
	ui.Blank()

	ui.StepStart("Checking token and world settings... ")
	settings, err := cl.GetRuntimeSettings(cmd.Context())
	if err != nil {
		ui.StepFail()
		if client.IsUnauthorized(err) {
			return fmt.Errorf("token invalid or expired — create a new token in the Developer section and run 'biterra config set checker_token <token>'")
		}
		return fmt.Errorf("could not reach the world API: %w", err)
	}
	round := settings.Round
	if round != nil {
		ui.StepOK(fmt.Sprintf("round %d active", round.RoundIndex))
	} else {
		ui.StepOK("no round active")
	}

	ui.Blank()
	ui.KeyValue("Token", "valid")
	if round != nil {
		ui.KeyValue("Round", fmt.Sprintf("%d", round.RoundIndex))
	} else {
		ui.KeyValue("Round", "(none)")
	}
	ui.KeyValue("Tick Interval", fmt.Sprintf("%ds", settings.TickIntervalSeconds))
	ui.Rule()
	return nil
}
