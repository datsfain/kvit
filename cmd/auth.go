package cmd

import (
	"fmt"
	"kvit/drive"
	"os"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Sign in with Google for Drive sync",
	Run: func(cmd *cobra.Command, args []string) {
		if !forceAuth && drive.IsAuthenticated() {
			fmt.Println("Already authenticated.")
			fmt.Println("Run 'kvit auth --force' to re-authenticate.")
			return
		}
		doAuth()
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored Google credentials",
	Run: func(cmd *cobra.Command, args []string) {
		if err := drive.Logout(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Logged out.")
	},
}

var forceAuth bool

func init() {
	authCmd.Flags().BoolVar(&forceAuth, "force", false, "Re-authenticate even if already signed in")
	authCmd.AddCommand(authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}

func doAuth() {
	if err := drive.Auth(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Authenticated! You can now use 'kvit sync push' and 'kvit sync pull'.")
}
