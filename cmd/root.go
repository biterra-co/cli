package cmd

import (
	"os"

	"github.com/biterra-co/cli/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "biterra",
	Short: "Biterra checker CLI for A/D game setup and validation",
	Long:  "Configure and validate access to the Checker API. Use 'biterra init' to set up, 'biterra check' to validate, 'biterra env' to export env for the checker process.",
}

func init() {
	defaultHelpFunc := rootCmd.HelpFunc()

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		ui.LogoPrint()
		ui.Blank()
		defaultHelpFunc(cmd, args)
	})
}

func Execute() {
	rootCmd.SilenceErrors = true
	if err := rootCmd.Execute(); err != nil {
		ui.ErrorToStderr(err.Error())
		os.Exit(1)
	}
}
