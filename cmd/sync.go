package cmd

import (
	"bufio"
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

Usage:
  kvit sync push     Upload local files to Drive (overwrites remote)
  kvit sync pull     Download files from Drive (overwrites local)`,
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload local CSV files to Google Drive",
	Run: func(cmd *cobra.Command, args []string) {
		requireReady()
		syncWithRetry("⬆ Pushing to Google Drive...", func() error {
			return drive.Push(config.SyncableFiles())
		})
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download CSV files from Google Drive",
	Run: func(cmd *cobra.Command, args []string) {
		requireReady()
		runPull()
	},
}

var syncLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link to a Google Drive folder",
	Long: `Link kvit to a Google Drive folder for syncing.

Create a folder on Google Drive, copy its URL, and paste it here.
For family sharing, the folder owner shares it and others link to the same folder.`,
	Run: func(cmd *cobra.Command, args []string) {
		requireAuth()
		promptAndLink(args)
	},
}

var syncOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the linked Google Drive folder in your browser",
	Run: func(cmd *cobra.Command, args []string) {
		requireReady()
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

// ── Shared helpers ──

func askYN(prompt string) bool {
	fmt.Print(prompt)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(input)
}

// requireAuth ensures the user is authenticated, offering to sign in if not
func requireAuth() {
	if !drive.IsAuthenticated() {
		if !askYN("Not authenticated. Sign in with Google now? [Y/n]: ") {
			fmt.Println("Cancelled.")
			os.Exit(0)
		}
		if err := drive.Auth(); err != nil {
			fmt.Fprintf(os.Stderr, "Auth failed: %v\n", err)
			os.Exit(1)
		}
	}
}

// requireReady ensures auth + folder linked, prompting interactively for each
func requireReady() {
	requireAuth()

	if !drive.IsFolderLinked() {
		fmt.Println("No Google Drive folder linked.")
		fmt.Println("Create a folder on Google Drive, then paste its URL here.")
		fmt.Println()
		input := readLine("Folder URL (or Enter to cancel): ")
		if input == "" {
			fmt.Println("Cancelled. Create a folder on Google Drive and run: kvit sync link <url>")
			os.Exit(0)
		}
		if err := drive.LinkFolder(input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func promptAndLink(args []string) {
	var input string
	if len(args) > 0 {
		input = args[0]
	} else {
		input = readLine("Paste the Google Drive folder URL: ")
	}
	if input == "" {
		fmt.Println("No URL provided.")
		return
	}
	if err := drive.LinkFolder(input); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func isAuthError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "401") ||
		strings.Contains(s, "Invalid Credentials") ||
		strings.Contains(s, "token expired") ||
		strings.Contains(s, "failed to refresh token") ||
		strings.Contains(s, "not authenticated")
}

func reauth() bool {
	if !askYN("\nAuthentication expired. Re-authenticate? [Y/n]: ") {
		fmt.Println("Cancelled.")
		return false
	}
	if err := drive.Auth(); err != nil {
		fmt.Fprintf(os.Stderr, "Auth failed: %v\n", err)
		return false
	}
	return true
}

func syncWithRetry(msg string, op func() error) {
	fmt.Println(msg)
	err := op()
	if err == nil {
		fmt.Println("Done.")
		return
	}
	if isAuthError(err) && reauth() {
		fmt.Println("\nRetrying...")
		if retryErr := op(); retryErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", retryErr)
			os.Exit(1)
		}
		fmt.Println("Done.")
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func runPull() {
	fmt.Println("⬇ Pulling from Google Drive...")
	n, err := drive.Pull(config.SyncableFiles())
	if err != nil {
		if isAuthError(err) && reauth() {
			fmt.Println("\nRetrying...")
			n, err = drive.Pull(config.SyncableFiles())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	if n == 0 {
		fmt.Println("Folder is empty, nothing to pull. Use kvit sync push to upload your data.")
	} else {
		fmt.Println("Done.")
	}
}
