package cmd

import (
	"fmt"
	"kvit/config"
	"kvit/drive"
	"os"
	"strings"

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

var syncLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link to a shared Google Drive folder",
	Long: `Link kvit to a shared Google Drive folder so multiple people can sync to the same data.

The folder owner runs "kvit sync open" and shares the folder. Others paste the folder URL here.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureAuth()

		if len(args) > 0 {
			if err := drive.LinkFolder(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Interactive: ask for URL
		fmt.Print("Paste the shared Google Drive folder URL: ")
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Println("No URL provided.")
			return
		}
		if err := drive.LinkFolder(input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
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
	syncCmd.AddCommand(syncLinkCmd)
	rootCmd.AddCommand(syncCmd)
}

func ensureAuth() {
	if !drive.IsAuthenticated() {
		fmt.Fprintln(os.Stderr, "Not authenticated. Run: kvit auth")
		os.Exit(1)
	}
}
