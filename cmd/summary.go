package cmd

import (
	"encoding/json"
	"fmt"
	"kvit/report"
	"kvit/storage"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate an interactive HTML expense report",
	Run:   runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

type reportData struct {
	Expenses    []expenseJSON    `json:"expenses"`
	Definitions []definitionJSON `json:"definitions"`
	Colors      map[string]string `json:"colors"`
}

type expenseJSON struct {
	Date    string  `json:"date"`
	Store   string  `json:"store"`
	Product string  `json:"product"`
	Price   float64 `json:"price"`
}

type definitionJSON struct {
	Product  string `json:"product"`
	Category string `json:"category"`
}

func runSummary(cmd *cobra.Command, args []string) {
	expenses, err := storage.LoadExpenses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading expenses: %v\n", err)
		os.Exit(1)
	}
	if len(expenses) == 0 {
		fmt.Println("No expenses found.")
		return
	}

	defs, _ := storage.LoadDefinitions()
	colors := storage.LoadColors()

	var data reportData
	data.Colors = colors
	for _, e := range expenses {
		data.Expenses = append(data.Expenses, expenseJSON{
			Date: e.Date, Store: e.Store, Product: e.Product, Price: e.Price,
		})
	}
	for _, d := range defs {
		data.Definitions = append(data.Definitions, definitionJSON{
			Product: d.Product, Category: d.Category,
		})
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding data: %v\n", err)
		os.Exit(1)
	}

	// Assemble HTML from template + CSS + JS + data
	html := report.TemplateHTML
	html = strings.Replace(html, "{{CSS}}", report.StyleCSS, 1)
	html = strings.Replace(html, "{{DATA}}", string(jsonBytes), 1)
	html = strings.Replace(html, "{{JS}}", report.AppJS, 1)

	outPath, _ := filepath.Abs("kvit-report.html")
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Report generated: %s\n", outPath)

	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.Command("open", outPath)
	case "linux":
		openCmd = exec.Command("xdg-open", outPath)
	}
	if openCmd != nil {
		openCmd.Start()
	}
}
