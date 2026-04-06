package cmd

import (
	"fmt"
	"kvit/config"
	"kvit/models"
	"kvit/report"
	"kvit/storage"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Generate and copy an AI prompt for receipt scanning",
	Run:   runPrompt,
}

func init() {
	rootCmd.AddCommand(promptCmd)
}

func runPrompt(cmd *cobra.Command, args []string) {
	products := storage.ProductNames()
	stores := storage.UniqueStores()
	exclusions := storage.LoadExclusions()

	var filteredProducts []string
	for _, p := range products {
		if !exclusions[p] {
			filteredProducts = append(filteredProducts, p)
		}
	}

	prompt := report.PromptTemplate
	prompt = strings.Replace(prompt, "{{TODAY}}", models.Today(), 1)
	prompt = strings.Replace(prompt, "{{PRODUCTS}}", strings.Join(filteredProducts, ", "), 1)
	prompt = strings.Replace(prompt, "{{STORES}}", strings.Join(stores, ", "), 1)
	prompt = strings.ReplaceAll(prompt, "{{CURRENCY}}", config.Currency())
	prompt = strings.Replace(prompt, "{{LANGUAGES}}", strings.Join(config.Languages(), ", "), 1)

	err := clipboard.WriteAll(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not copy to clipboard: %v\n", err)
		fmt.Fprintf(os.Stderr, "Install xclip (X11) or wl-clipboard (Wayland) for clipboard support.\n\n")
	} else {
		fmt.Println(successStyle.Render("✓ Prompt copied to clipboard!"))
		fmt.Println()
	}

	fmt.Println(headerStyle.Render("─── AI Prompt ───"))
	fmt.Println()
	fmt.Println(prompt)
	fmt.Println()
	fmt.Printf(itemStyle.Render("Known products: %d (excluded: %d) | Known stores: %d\n"),
		len(filteredProducts), len(exclusions), len(stores))
}
