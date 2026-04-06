package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

// installCompletions writes shell completions to the system directory.
// Called after install/update so tab-completion works automatically.
func installCompletions() {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return
	}

	dirs := []string{"/usr/share/bash-completion/completions", "/etc/bash_completion.d"}
	var dir string
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			dir = d
			break
		}
	}
	if dir == "" {
		return
	}

	path := filepath.Join(dir, "kvit")
	// Try direct write first, fall back to sudo
	if err := os.WriteFile(path, []byte(completionScript()), 0644); err != nil {
		cmd := exec.Command("sudo", "tee", path)
		cmd.Stdin = strings.NewReader(completionScript())
		cmd.Stdout = nil
		cmd.Run()
	}
}

func completionScript() string {
	buf := new(strings.Builder)
	rootCmd.GenBashCompletionV2(buf, true)
	return buf.String()
}

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

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating updater: %v\n", err)
		os.Exit(1)
	}

	latest, found, err := updater.DetectLatest(cmd.Context(), selfupdate.ParseSlug("datsfain/kvit"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}
	if !found {
		fmt.Println("No releases found.")
		return
	}

	if Version != "dev" && latest.LessOrEqual(Version) {
		fmt.Printf("Already up to date (latest: %s)\n", latest.Version())
		return
	}

	fmt.Printf("Updating to %s...\n", latest.Version())

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
		os.Exit(1)
	}

	if err := updater.UpdateTo(cmd.Context(), latest, exe); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Error: permission denied. Try: sudo kvit update\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
		}
		os.Exit(1)
	}

	installCompletions()
	fmt.Printf("✓ Updated to %s (checksum verified)\n", latest.Version())
}
