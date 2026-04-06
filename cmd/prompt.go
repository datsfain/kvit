package cmd

import (
	"fmt"
	"kvit/models"
	"kvit/storage"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Generate and copy a Gemini prompt for receipt scanning",
	Run:   runPrompt,
}

func init() {
	rootCmd.AddCommand(promptCmd)
}

func runPrompt(cmd *cobra.Command, args []string) {
	products := storage.ProductNames()
	stores := storage.UniqueStores()
	exclusions := storage.LoadExclusions()

	// Filter out excluded products
	var filteredProducts []string
	for _, p := range products {
		if !exclusions[p] {
			filteredProducts = append(filteredProducts, p)
		}
	}

	today := models.Today()

	prompt := fmt.Sprintf(`You are a receipt parser. I will give you one or more photos of shopping receipts (in Danish). Your job is to extract products and prices and generate a CLI command.

## Command format

%s

- Store name: lowercase, hyphenated (e.g. netto, super-brugsen)
- Date: YYYY-MM-DD format. Extract from receipt if visible, otherwise use %s
- Product names: lowercase, hyphenated English names (e.g. ground-beef, chicken-breast)
- Prices: in DKK, decimal or whole number
- If multiple receipts from different stores, separate with +

## Known products (use these names when possible)

%s

## Known stores

%s

## Rules

1. ALWAYS prefer existing product names from the list above. For example, if the receipt says "kyllingebryst" (Danish), use "chicken-breast" since it already exists.
2. For new products not in the list, create a descriptive hyphenated English name that fits the existing naming style.
3. Ignore non-product lines: VAT, totals, subtotals, payment method, change, card numbers, loyalty points, bag fees.
4. If a discount applies to a specific product, subtract it from that product's price (don't add a separate discount line).
5. If a receipt shows quantity × unit price, calculate the total price for that line.
6. Prices should be the final price paid for each item.

## Output format

First, output a human-readable table so I can verify:

%s

Then, output the command as a copiable code block:

%s

Always output BOTH formats so I can verify before running the command.`,
		"```\nkvit add <store> [YYYY-MM-DD] <product:price> [product:price...] [+ <store> [YYYY-MM-DD] <product:price>...]\n```",
		today,
		strings.Join(filteredProducts, ", "),
		strings.Join(stores, ", "),
		"```\n| Product        | Price (DKK) | Matched to       |\n|----------------|-------------|------------------|\n| Kyllingebryst  | 45.00       | chicken-breast   |\n| Agurk          | 12.00       | cucumber         |\n| Nye varer      | 30.00       | new-product-name |\n```",
		"```\nkvit add netto 2026-04-06 chicken-breast:45 cucumber:12 new-product-name:30\n```",
	)

	// Try to copy to clipboard
	err := clipboard.WriteAll(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not copy to clipboard: %v\n", err)
		fmt.Fprintf(os.Stderr, "Install xclip (X11) or wl-clipboard (Wayland) for clipboard support.\n\n")
	} else {
		fmt.Println(successStyle.Render("✓ Prompt copied to clipboard!"))
		fmt.Println()
	}

	// Always print the prompt too
	fmt.Println(headerStyle.Render("─── Gemini Prompt ───"))
	fmt.Println()
	fmt.Println(prompt)
	fmt.Println()
	fmt.Printf(itemStyle.Render("Known products: %d (excluded: %d) | Known stores: %d\n"),
		len(filteredProducts), len(exclusions), len(stores))
}
