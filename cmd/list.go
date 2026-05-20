package cmd

import (
	"fmt"
	"kvit/storage"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "products · categories — list known products and categories",
}

var listProductsCmd = &cobra.Command{
	Use:   "products",
	Short: "List all known products with their category",
	Run: func(cmd *cobra.Command, args []string) {
		contains, _ := cmd.Flags().GetString("contains")
		startsWith, _ := cmd.Flags().GetString("starts-with")

		defs, err := storage.LoadDefinitions()
		if err != nil {
			fmt.Println("Error loading definitions:", err)
			return
		}

		// measure longest product name for alignment
		maxLen := 0
		for _, d := range defs {
			if matches(d.Product, contains, startsWith) && len(d.Product) > maxLen {
				maxLen = len(d.Product)
			}
		}

		count := 0
		for _, d := range defs {
			if matches(d.Product, contains, startsWith) {
				fmt.Printf("%-*s  %s\n", maxLen, d.Product, d.Category)
				count++
			}
		}
		if count == 0 {
			fmt.Println("No products found.")
		}
	},
}

var listCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List all known categories",
	Run: func(cmd *cobra.Command, args []string) {
		contains, _ := cmd.Flags().GetString("contains")
		startsWith, _ := cmd.Flags().GetString("starts-with")

		defs, err := storage.LoadDefinitions()
		if err != nil {
			fmt.Println("Error loading definitions:", err)
			return
		}

		seen := make(map[string]int)
		order := []string{}
		for _, d := range defs {
			if _, ok := seen[d.Category]; !ok {
				order = append(order, d.Category)
			}
			seen[d.Category]++
		}

		count := 0
		for _, cat := range order {
			if matches(cat, contains, startsWith) {
				fmt.Printf("%s  (%d products)\n", cat, seen[cat])
				count++
			}
		}
		if count == 0 {
			fmt.Println("No categories found.")
		}
	},
}

func init() {
	listProductsCmd.Flags().String("contains", "", "Filter by substring")
	listProductsCmd.Flags().String("starts-with", "", "Filter by prefix")
	listCategoriesCmd.Flags().String("contains", "", "Filter by substring")
	listCategoriesCmd.Flags().String("starts-with", "", "Filter by prefix")

	listCmd.AddCommand(listProductsCmd)
	listCmd.AddCommand(listCategoriesCmd)
	rootCmd.AddCommand(listCmd)
}

func matches(name, contains, startsWith string) bool {
	lower := strings.ToLower(name)
	if contains != "" && !strings.Contains(lower, strings.ToLower(contains)) {
		return false
	}
	if startsWith != "" && !strings.HasPrefix(lower, strings.ToLower(startsWith)) {
		return false
	}
	return true
}
