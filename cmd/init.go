package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/geoctf/biterra-cli/internal/client"
	"github.com/geoctf/biterra-cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup: API URL, token, team, and service",
	Long:  "Prompts for API base URL and checker token, validates with the API, then prompts to pick team and service. Saves config to file.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.Load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	reader := bufio.NewReader(os.Stdin)

	// API URL
	if cfg.APIURL == "" {
		cfg.APIURL = os.Getenv("BITERRA_API_URL")
	}
	if cfg.APIURL == "" {
		fmt.Print("API base URL (e.g. https://world.example.com): ")
		line, _ := reader.ReadString('\n')
		cfg.APIURL = strings.TrimSpace(line)
		if cfg.APIURL == "" {
			return fmt.Errorf("api_url is required")
		}
	}
	cfg.APIURL = strings.TrimSuffix(cfg.APIURL, "/")

	// Token
	if cfg.CheckerToken == "" {
		cfg.CheckerToken = os.Getenv("BITERRA_CHECKER_TOKEN")
	}
	if cfg.CheckerToken == "" {
		fmt.Print("Checker token (from portal: A/D tab → Rotate token): ")
		line, _ := reader.ReadString('\n')
		cfg.CheckerToken = strings.TrimSpace(line)
		if cfg.CheckerToken == "" {
			return fmt.Errorf("checker_token is required")
		}
	}

	// Validate token
	cl := client.New(cfg.APIURL, cfg.CheckerToken)
	round, err := cl.GetRoundsCurrent(cmd.Context())
	if err != nil {
		if client.IsUnauthorized(err) {
			return fmt.Errorf("invalid or expired checker token: %w", err)
		}
		return fmt.Errorf("validate token: %w", err)
	}
	if round != nil {
		fmt.Printf("Token valid. Current round: index=%d\n", round.RoundIndex)
	} else {
		fmt.Println("Token valid. No round currently active.")
	}

	// Teams
	teams, err := cl.GetTeams(cmd.Context())
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}
	if len(teams) > 0 && cfg.TeamUID == "" {
		fmt.Println("Teams:")
		for i, t := range teams {
			fmt.Printf("  %d) %s (%s)\n", i+1, t.Name, t.UID)
		}
		fmt.Print("Which team UID (or number 1-N): ")
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)
		if choice != "" {
			if n := parseNumber(choice); n > 0 && n <= len(teams) {
				cfg.TeamUID = teams[n-1].UID
			} else {
				cfg.TeamUID = choice
			}
		}
	}

	// Services
	services, err := cl.GetServices(cmd.Context(), "")
	if err != nil {
		return fmt.Errorf("list services: %w", err)
	}
	if len(services) > 0 && cfg.ServiceUID == "" {
		fmt.Println("Services:")
		for i, s := range services {
			ri := ""
			if s.RoundIndex != nil {
				ri = fmt.Sprintf(" round %d", *s.RoundIndex)
			}
			fmt.Printf("  %d) %s (%s)%s\n", i+1, s.Name, s.UID, ri)
		}
		fmt.Print("Which service UID (or number 1-N): ")
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)
		if choice != "" {
			if n := parseNumber(choice); n > 0 && n <= len(services) {
				cfg.ServiceUID = services[n-1].UID
			} else {
				cfg.ServiceUID = choice
			}
		}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Println("Config saved.")
	return nil
}

func parseNumber(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
