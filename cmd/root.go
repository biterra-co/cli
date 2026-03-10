package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "biterra",
	Short: "Biterra checker CLI for A/D game setup and validation",
	Long:  "Configure and validate access to the Checker API. Use 'biterra init' to set up, 'biterra check' to validate, 'biterra env' to export env for the checker process.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra prints the error; exit non-zero
		// os.Exit(1) is handled by cobra when RunE returns err
	}
}
