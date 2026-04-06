package cmd

import (
	"fmt"
	"kvit/config"
	"kvit/drive"
	"os"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync CSV files with Google Drive",
	Long: `Sync your expense data with Google Drive.

Setup:
  kvit auth          Sign in with Google

Usage:
  kvit sync push     Upload local files to Drive (overwrites remote)
  kvit sync pull     Download files from Drive (overwrites local)`,
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload local CSV files to Google Drive",
	Run: func(cmd *cobra.Command, args []string) {
		ensureAuth()
		fmt.Println("⬆ Pushing to Google Drive...")
		if err := drive.Push(config.SyncableFiles()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Done.")
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download CSV files from Google Drive",
	Run: func(cmd *cobra.Command, args []string) {
		ensureAuth()
		fmt.Println("⬇ Pulling from Google Drive...")
		if err := drive.Pull(config.SyncableFiles()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Done.")
	},
}

var syncOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the kvit folder on Google Drive in your browser",
	Run: func(cmd *cobra.Command, args []string) {
		ensureAuth()
		if err := drive.OpenFolder(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncOpenCmd)
	rootCmd.AddCommand(syncCmd)
}

func ensureAuth() {
	if !drive.IsAuthenticated() {
		fmt.Fprintln(os.Stderr, "Not authenticated. Run: kvit auth")
		os.Exit(1)
	}
}
