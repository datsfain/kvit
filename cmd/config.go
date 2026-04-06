package cmd

import (
	"fmt"
	"kvit/config"
	"kvit/drive"
	"strings"

	"github.com/spf13/cobra"
)

var configSetup bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or change kvit settings",
	Run:   runConfig,
}

func init() {
	configCmd.Flags().BoolVar(&configSetup, "setup", false, "Re-run initial setup")
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) {
	if configSetup {
		RunSetup()
		return
	}

	c := config.Load()

	fmt.Println(headerStyle.Render("─── kvit config ───"))
	fmt.Println()

	if c.Currency != "" {
		fmt.Printf("  Currency:   %s\n", c.Currency)
	} else {
		fmt.Printf("  Currency:   %s\n", hintStyle.Render("not set"))
	}

	if len(c.Languages) > 0 {
		fmt.Printf("  Languages:  %s\n", strings.Join(c.Languages, ", "))
	} else {
		fmt.Printf("  Languages:  %s\n", hintStyle.Render("not set"))
	}

	if drive.IsFolderLinked() {
		fmt.Printf("  Drive:      %s\n", successStyle.Render("linked"))
	} else {
		fmt.Printf("  Drive:      %s\n", hintStyle.Render("not linked"))
	}

	if drive.IsAuthenticated() {
		fmt.Printf("  Auth:       %s\n", successStyle.Render("signed in"))
	} else {
		fmt.Printf("  Auth:       %s\n", hintStyle.Render("not signed in"))
	}

	fmt.Println()
	fmt.Println(hintStyle.Render("  Run 'kvit config --setup' to change currency and languages."))
}
