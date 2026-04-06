package cmd

import (
	"fmt"
	"kvit/models"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var productPriceRegex = regexp.MustCompile(`^([a-zA-Z0-9æøåÆØÅ][a-zA-Z0-9æøåÆØÅ\-]*):(\d+\.?\d*)$`)

var addCmd = &cobra.Command{
	Use:   "add <store> [YYYY-MM-DD] <product:price>... [+ <store> [YYYY-MM-DD] <product:price>...]",
	Short: "Add expenses from a one-liner command",
	Long: `Add expenses for one or more stores.

Examples:
  kvit add netto ground-beef:200 cucumber:30
  kvit add netto 2026-04-05 ground-beef:200 cucumber:30
  kvit add netto ground-beef:200 + føtex milk:12.50 bread:25`,
	Args: cobra.MinimumNArgs(2),
	Run:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	entries, err := parseAddArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if ConfirmAndSave(entries) {
		HandleUnknownProducts(entries)
	}
}

func parseAddArgs(args []string) ([]models.StoreEntry, error) {
	// Split args by "+"
	groups := splitByPlus(args)

	var entries []models.StoreEntry
	defaultDate := models.Today()

	for _, group := range groups {
		if len(group) < 2 {
			return nil, fmt.Errorf("each store group needs at least a store name and one product:price")
		}

		entry := models.StoreEntry{}
		idx := 0

		// First arg is store
		entry.Store = strings.ToLower(group[idx])
		idx++

		// Check if next arg is a date
		if idx < len(group) && dateRegex.MatchString(group[idx]) {
			entry.Date = group[idx]
			idx++
		} else {
			entry.Date = defaultDate
		}

		// Rest are product:price pairs
		for ; idx < len(group); idx++ {
			match := productPriceRegex.FindStringSubmatch(group[idx])
			if match == nil {
				return nil, fmt.Errorf("invalid product:price format: %q (expected name:price, e.g. ground-beef:200)", group[idx])
			}
			price, err := strconv.ParseFloat(match[2], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid price in %q: %v", group[idx], err)
			}
			entry.Products = append(entry.Products, models.ProductPrice{
				Product: strings.ToLower(match[1]),
				Price:   price,
			})
		}

		if len(entry.Products) == 0 {
			return nil, fmt.Errorf("no products specified for store %q", entry.Store)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func splitByPlus(args []string) [][]string {
	var groups [][]string
	var current []string

	for _, arg := range args {
		if arg == "+" {
			if len(current) > 0 {
				groups = append(groups, current)
			}
			current = nil
		} else {
			current = append(current, arg)
		}
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}
