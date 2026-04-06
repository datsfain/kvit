package cmd

import (
	"fmt"
	"os"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kvit to the latest version",
	Run:   runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	latest, found, err := selfupdate.DetectLatest(cmd.Context(), selfupdate.ParseSlug("datsfain/kvit"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}
	if !found {
		fmt.Println("No releases found.")
		return
	}

	if latest.LessOrEqual(Version) {
		fmt.Printf("Already up to date (latest: %s)\n", latest.Version())
		return
	}

	fmt.Printf("Updating to %s...\n", latest.Version())

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
		os.Exit(1)
	}

	if err := selfupdate.UpdateTo(cmd.Context(), latest.AssetURL, latest.AssetName, exe); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Error: permission denied. Try: sudo kvit update\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Updated to %s\n", latest.Version())
}
