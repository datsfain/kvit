package cmd

import (
	"fmt"
	"kvit/models"
	"kvit/storage"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Confirm TUI ---

type confirmModel struct {
	entries    []models.StoreEntry
	textInput  textinput.Model
	confirmed  bool
	done       bool
}

func newConfirmModel(entries []models.StoreEntry) confirmModel {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = labelStyle.Render("Save? [Y/n]: ")
	ti.CharLimit = 3

	return confirmModel{
		entries:   entries,
		textInput: ti,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		case "enter":
			answer := strings.TrimSpace(strings.ToLower(m.textInput.Value()))
			m.confirmed = answer == "" || answer == "y" || answer == "yes"
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n" + headerStyle.Render("── Confirm ──────────────────────────────") + "\n")

	grandTotal := 0.0
	for _, entry := range m.entries {
		sorted := make([]models.ProductPrice, len(entry.Products))
		copy(sorted, entry.Products)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Price > sorted[j].Price
		})

		b.WriteString("\n" + storeStyle.Render(fmt.Sprintf("  %s (%s):", entry.Store, entry.Date)) + "\n")
		subtotal := 0.0
		for _, p := range sorted {
			b.WriteString(fmt.Sprintf("    %-20s", p.Product) +
				priceStyle.Render(fmt.Sprintf("%10.2f DKK", p.Price)) + "\n")
			subtotal += p.Price
		}
		b.WriteString(totalStyle.Render(fmt.Sprintf("    %-20s %10.2f DKK", "Subtotal:", subtotal)) + "\n")
		grandTotal += subtotal
	}

	b.WriteString("\n" + totalStyle.Render(fmt.Sprintf("  %-22s %10.2f DKK", "Total:", grandTotal)) + "\n")
	b.WriteString(headerStyle.Render("──────────────────────────────────────────") + "\n\n")

	b.WriteString(m.textInput.View())

	return b.String()
}

// ConfirmAndSave shows a confirmation dialog and saves if user confirms.
func ConfirmAndSave(entries []models.StoreEntry) bool {
	m := newConfirmModel(entries)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return false
	}

	result := finalModel.(confirmModel)
	if !result.confirmed {
		fmt.Println("Cancelled.")
		return false
	}

	var allExpenses []models.Expense
	for _, entry := range entries {
		for _, p := range entry.Products {
			allExpenses = append(allExpenses, models.Expense{
				Date:    entry.Date,
				Store:   entry.Store,
				Product: p.Product,
				Price:   p.Price,
			})
		}
	}

	if err := storage.AppendExpenses(allExpenses); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
		return false
	}

	fmt.Printf("✓ Saved %d items.\n", len(allExpenses))
	return true
}

// --- Categorization TUI ---

type catModel struct {
	unknown    []string
	categories []string
	results    []models.Definition
	current    int
	textInput  textinput.Model
	done       bool
	cancelled  bool
}

func newCatModel(unknown []string, categories []string) catModel {
	ti := textinput.New()
	ti.Focus()
	ti.ShowSuggestions = true
	ti.SetSuggestions(categories)
	ti.Prompt = labelStyle.Render(fmt.Sprintf("  %s → ", unknown[0]))

	return catModel{
		unknown:    unknown,
		categories: categories,
		current:    0,
		textInput:  ti,
	}
}

func (m catModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m catModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit

		case "enter":
			value := strings.TrimSpace(m.textInput.Value())

			if value != "" {
				m.results = append(m.results, models.Definition{
					Product:  m.unknown[m.current],
					Category: value,
				})

				// Add new category to suggestions immediately
				found := false
				for _, c := range m.categories {
					if c == value {
						found = true
						break
					}
				}
				if !found {
					m.categories = append(m.categories, value)
				}
			}

			m.current++
			if m.current >= len(m.unknown) {
				m.done = true
				return m, tea.Quit
			}

			m.textInput.Reset()
			m.textInput.SetSuggestions(m.categories)
			m.textInput.Prompt = labelStyle.Render(fmt.Sprintf("  %s → ", m.unknown[m.current]))
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m catModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString(warnStyle.Render("⚠ Categorize new products:") + "\n\n")

	// Show already categorized
	for _, d := range m.results {
		b.WriteString(itemStyle.Render(fmt.Sprintf("  %-20s → ", d.Product)) +
			successStyle.Render(d.Category) + "\n")
	}

	// Show remaining
	for i := m.current + 1; i < len(m.unknown); i++ {
		b.WriteString(itemStyle.Render(fmt.Sprintf("  %-20s   ?", m.unknown[i])) + "\n")
	}

	b.WriteString("\n")

	// Show existing categories as hint
	if len(m.categories) > 0 {
		b.WriteString(hintStyle.Render("  Categories: "+strings.Join(m.categories, ", ")) + "\n")
	}

	b.WriteString(m.textInput.View())

	return b.String()
}

// HandleUnknownProducts checks for products not in definitions and prompts for categorization
func HandleUnknownProducts(entries []models.StoreEntry) {
	var unknown []string
	seen := make(map[string]bool)

	for _, entry := range entries {
		for _, p := range entry.Products {
			if !seen[p.Product] && !storage.IsKnownProduct(p.Product) {
				unknown = append(unknown, p.Product)
				seen[p.Product] = true
			}
		}
	}

	if len(unknown) == 0 {
		return
	}

	categories := storage.CategoryNames()
	m := newCatModel(unknown, categories)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	result := finalModel.(catModel)

	if result.cancelled || len(result.results) == 0 {
		return
	}

	if err := storage.AppendDefinitions(result.results); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving definitions: %v\n", err)
		return
	}
	fmt.Printf("✓ Definitions updated (%d new products).\n", len(result.results))
}
