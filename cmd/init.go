package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/biterra-co/cli/internal/browser"
	"github.com/biterra-co/cli/internal/client"
	"github.com/biterra-co/cli/internal/config"
	"github.com/biterra-co/cli/internal/discovery"
	"github.com/biterra-co/cli/internal/ui"
	"github.com/spf13/cobra"
)

// developerPath is the customer portal path for creating checker tokens (Account → Developer).
const developerPath = "/settings/account#developer"

// defaultCustomerPortalURL is used when opening the browser for token setup and for token-info lookup.
const defaultCustomerPortalURL = "https://ctf.biterra.co"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup: token, team, and service",
	Long:  "Prompts for your checker token (create one in the customer portal → Developer), looks up which world it's for, validates, then prompts to pick team and service.",
	RunE:  runInit,
	// Don't print full usage on every error; our errors are self-explanatory.
	SilenceUsage: true,
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

	customerPortalURL := cfg.CustomerPortalURL
	if customerPortalURL == "" {
		customerPortalURL = os.Getenv("BITERRA_CUSTOMER_PORTAL_URL")
	}
	if customerPortalURL == "" {
		customerPortalURL = defaultCustomerPortalURL
	}
	customerPortalURL = strings.TrimSuffix(customerPortalURL, "/")

	// Token: single prompt — paste token or press Enter to open browser (or keep current if set)
	if cfg.CheckerToken == "" {
		cfg.CheckerToken = os.Getenv("BITERRA_CHECKER_TOKEN")
	}
	ui.Header("Biterra Checker Setup", "We'll need your checker token.")
	ui.Blank()
	ui.Rule()
	ui.Blank()
	portalURL := customerPortalURL + developerPath
	if cfg.CheckerToken != "" {
		ui.Muted("  Paste a new token below, or press Enter to keep your current token.")
	} else {
		ui.Muted("  Paste your checker token below, or press Enter to open the browser and create one.")
	}
	ui.Blank()
	ui.Prompt("Checker token: ")
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		cfg.CheckerToken = line
		cfg.APIURL = "" // force lookup for new token
	} else if cfg.CheckerToken == "" {
		// Empty and no current token — open browser and prompt again
		if err := browser.Open(portalURL); err == nil {
			ui.Info("Browser opened. Sign in and create a token in the Developer section.")
		} else {
			ui.Muted("Open this URL in your browser:")
			ui.URL(portalURL)
		}
		ui.Blank()
		ui.Muted("  Then paste the token below.")
		ui.Prompt("Checker token: ")
		line, _ = reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			cfg.CheckerToken = line
			cfg.APIURL = ""
		}
	}
	if cfg.CheckerToken == "" {
		return fmt.Errorf("a checker token is required — create one in the Developer section and paste it here")
	}
	ui.Blank()

	// Resolve world API URL from token (if not already set)
	if cfg.APIURL == "" {
		cfg.APIURL = os.Getenv("BITERRA_API_URL")
	}
	if cfg.APIURL == "" {
		ui.Rule()
		ui.Blank()
		ui.StepStart("Looking up world for this token... ")
		apiURL, err := discovery.TokenInfo(cmd.Context(), customerPortalURL, cfg.CheckerToken)
		if err != nil {
			var portalErr *discovery.ErrPortalNotReady
			if errors.As(err, &portalErr) {
				ui.StepFail()
				ui.ErrorBlock(err.Error(), []string{
					fmt.Sprintf("Ensure the customer portal is running at %s", customerPortalURL),
					"For local dev, start the portal (e.g. yarn dev in next-customer-portal)",
					"Or set BITERRA_API_URL and skip lookup: biterra config set --api-url <world-api-url>",
				})
				return err
			}
			return err
		}
		ui.StepOK("")
		if apiURL == "" {
			apiURL = "https://world.biterra.co"
		}
		cfg.APIURL = apiURL
		ui.KeyValue("World API", cfg.APIURL)
	}
	cfg.APIURL = strings.TrimSuffix(cfg.APIURL, "/")

	// Validate token
	ui.Blank()
	ui.Rule()
	ui.Blank()
	ui.StepStart("Verifying token... ")
	cl := client.New(cfg.APIURL, cfg.CheckerToken)
	round, err := cl.GetRoundsCurrent(cmd.Context())
	if err != nil {
		ui.StepFail()
		if client.IsUnauthorized(err) {
			return fmt.Errorf("invalid or expired token — create a new one in the Developer section and run init again")
		}
		return fmt.Errorf("could not reach the world API: %w", err)
	}
	if round != nil {
		ui.StepOK(fmt.Sprintf("round %d", round.RoundIndex))
	} else {
		ui.StepOK("no round active")
	}
	ui.Blank()
	ui.Rule()
	ui.Blank()

	// Teams: always show so re-runs can override
	teams, err := cl.GetTeams(cmd.Context())
	if err != nil {
		return fmt.Errorf("could not load teams: %w", err)
	}
	if len(teams) > 0 {
		ui.Section("Teams")
		for i, t := range teams {
			ui.Option(i+1, t.Name, t.UID)
		}
		if cfg.TeamUID != "" {
			ui.Prompt("Select team (number or UID, or Enter to keep current): ")
		} else {
			ui.Prompt("Select team (number or UID): ")
		}
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

	// Services: always show so re-runs can override
	services, err := cl.GetServices(cmd.Context(), "")
	if err != nil {
		return fmt.Errorf("could not load services: %w", err)
	}
	if len(services) > 0 {
		ui.Section("Services")
		for i, s := range services {
			detail := s.UID
			if s.RoundIndex != nil {
				detail = fmt.Sprintf("%s, round %d", s.UID, *s.RoundIndex)
			}
			ui.Option(i+1, s.Name, detail)
		}
		if cfg.ServiceUID != "" {
			ui.Prompt("Select service (number or UID, or Enter to keep current): ")
		} else {
			ui.Prompt("Select service (number or UID): ")
		}
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

	// Register checker instance now so run can focus on checks/SLA.
	if strings.TrimSpace(cfg.TeamUID) != "" && strings.TrimSpace(cfg.ServiceUID) != "" {
		ui.Blank()
		ui.Rule()
		ui.Blank()
		ui.StepStart("Registering checker instance (team+service)... ")
		if _, err := resolveServiceRoundUID(cmd.Context(), cl, cfg.ServiceUID); err != nil {
			ui.StepFail()
			return fmt.Errorf("service validation failed: %w", err)
		}
		err := cl.PutTeamInstances(cmd.Context(), []client.TeamInstanceInput{
			{TeamUID: cfg.TeamUID, ServiceUID: cfg.ServiceUID},
		})
		if err != nil {
			ui.StepFail()
			if client.IsUnauthorized(err) {
				return fmt.Errorf("invalid or expired token — create a new one in the Developer section and run init again")
			}
			return fmt.Errorf("checker check-in failed: %w", err)
		}
		ui.StepOK("registered")
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}
	ui.SuccessBlock("Setup complete. Config saved.", []string{
		"biterra check  — verify token and see current round",
		"biterra env    — export variables for your checker",
	})
	return nil
}

func parseNumber(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
