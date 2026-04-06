package cmd

import (
	"fmt"
	"kvit/config"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync CSV files with Google Drive via rclone",
	Long: `Sync your expense data with Google Drive using rclone.

Setup:
  1. Install rclone: https://rclone.org/install/
  2. Configure rclone: rclone config
  3. Set remote: kvit config set remote "gdrive:expense-tracker"

Usage:
  kvit sync push   Upload local files to remote (overwrites remote)
  kvit sync pull   Download remote files to local (overwrites local)`,
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload local CSV files to remote",
	Run: func(cmd *cobra.Command, args []string) {
		ensureSync()
		fmt.Println("⬆ Pushing to remote...")
		rcloneSync("push")
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download remote CSV files to local",
	Run: func(cmd *cobra.Command, args []string) {
		ensureSync()
		fmt.Println("⬇ Pulling from remote...")
		rcloneSync("pull")
	},
}

func init() {
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	rootCmd.AddCommand(syncCmd)
}

func ensureSync() {
	// Check rclone is installed
	if _, err := exec.LookPath("rclone"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: rclone not found.")
		fmt.Fprintln(os.Stderr, "Install it: https://rclone.org/install/")
		fmt.Fprintln(os.Stderr, "  macOS:  brew install rclone")
		fmt.Fprintln(os.Stderr, "  Linux:  sudo apt install rclone")
		os.Exit(1)
	}

	// Check remote is configured
	remote, _ := config.GetSetting("remote")
	if remote == "" {
		fmt.Fprintln(os.Stderr, "Error: remote not configured.")
		fmt.Fprintln(os.Stderr, "Run: kvit config set remote \"gdrive:expense-tracker\"")
		os.Exit(1)
	}
}

func rcloneSync(direction string) bool {
	remote, _ := config.GetSetting("remote")
	files := config.SyncableFiles()

	allOk := true
	for _, file := range files {
		var args []string
		if direction == "push" {
			// Copy local file to remote
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue // skip files that don't exist locally
			}
			args = []string{"copyto", file, remote + "/" + file}
		} else {
			// Copy remote file to local
			args = []string{"copyto", remote + "/" + file, file}
		}

		cmd := exec.Command("rclone", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", file, err)
			allOk = false
		} else {
			fmt.Printf("  ✓ %s\n", file)
		}
	}

	if allOk {
		fmt.Println("Done.")
	}
	return allOk
}
