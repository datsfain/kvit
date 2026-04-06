package cmd

import (
	"fmt"
	"kvit/config"
	"os"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage kvit settings",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long:  "Available settings: remote (rclone remote path, e.g. gdrive:expense-tracker)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.SetSetting(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ %s = %s\n", args[0], args[1])
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		val, err := config.GetSetting(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if val == "" {
			fmt.Println("(not set)")
		} else {
			fmt.Println(val)
		}
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all settings",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := config.LoadSettings()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		remote := s.Remote
		if remote == "" {
			remote = "(not set)"
		}
		fmt.Printf("remote: %s\n", remote)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
