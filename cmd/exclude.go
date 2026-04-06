package cmd

import (
	"fmt"
	"kvit/storage"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var excludeCmd = &cobra.Command{
	Use:   "exclude",
	Short: "Manage product exclusions from the Gemini prompt",
}

var excludeAddCmd = &cobra.Command{
	Use:   "add <product> [product...]",
	Short: "Exclude products from the Gemini prompt",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exclusions := storage.LoadExclusions()
		for _, product := range args {
			product = strings.ToLower(product)
			if exclusions != nil && exclusions[product] {
				fmt.Printf("  %s already excluded\n", product)
				continue
			}
			if err := storage.AddExclusion(product); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			fmt.Printf("  ✓ %s excluded\n", product)
		}
	},
}

var excludeRemoveCmd = &cobra.Command{
	Use:   "remove <product> [product...]",
	Short: "Remove products from the exclusion list",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, product := range args {
			product = strings.ToLower(product)
			if err := storage.RemoveExclusion(product); err != nil {
				fmt.Fprintf(os.Stderr, "  %v\n", err)
				continue
			}
			fmt.Printf("  ✓ %s removed from exclusions\n", product)
		}
	},
}

var excludeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List excluded products",
	Run: func(cmd *cobra.Command, args []string) {
		exclusions := storage.LoadExclusions()
		if len(exclusions) == 0 {
			fmt.Println("No exclusions.")
			return
		}
		fmt.Println("Excluded products:")
		for p := range exclusions {
			fmt.Printf("  - %s\n", p)
		}
	},
}

func init() {
	excludeCmd.AddCommand(excludeAddCmd)
	excludeCmd.AddCommand(excludeRemoveCmd)
	excludeCmd.AddCommand(excludeListCmd)
	rootCmd.AddCommand(excludeCmd)
}
