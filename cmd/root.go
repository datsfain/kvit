package cmd

import (
	"fmt"
	"kvit/config"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "kvit",
	Short:   "kvit — a simple expense tracker",
	Long:    "Track daily expenses with ease. Store data in CSV files for analysis in Google Sheets, Grafana, or any tool.",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip setup for commands that don't need it
		name := cmd.Name()
		if name == "help" || name == "version" || name == "config" {
			return
		}
		if !config.IsConfigured() {
			if !RunSetup() {
				os.Exit(0)
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
