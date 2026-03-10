package cmd

import (
	"fmt"
	"os"

	"github.com/geoctf/biterra-cli/internal/client"
	"github.com/geoctf/biterra-cli/internal/config"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate token: call GET /rounds/current",
	Long:  "Uses stored config to call the checker API. Exits 0 if token is valid, 1 with message otherwise.",
	RunE:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRequired()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no config: run 'biterra init' or set BITERRA_API_URL and BITERRA_CHECKER_TOKEN")
		}
		return err
	}
	cl := client.New(cfg.APIURL, cfg.CheckerToken)
	round, err := cl.GetRoundsCurrent(cmd.Context())
	if err != nil {
		if client.IsUnauthorized(err) {
			return fmt.Errorf("invalid or expired checker token. Rotate the token in the world portal and run 'biterra config set --token NEW_TOKEN'")
		}
		return err
	}
	if round != nil {
		fmt.Printf("OK — current round: index=%d\n", round.RoundIndex)
	} else {
		fmt.Println("OK — no round currently active")
	}
	return nil
}
