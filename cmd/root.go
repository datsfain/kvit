package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kvit",
	Short: "kvit — a simple expense tracker",
	Long:  "Track daily expenses with ease. Store data in CSV files for analysis in Google Sheets, Grafana, or any tool.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
