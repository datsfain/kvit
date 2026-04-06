package cmd

import (
	"fmt"
	"kvit/config"
	"kvit/models"
	"kvit/storage"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Short:   "Interactively add expenses with autocomplete",
	Aliases: []string{"i"},
	Run:     runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

// Phase of the interactive flow
type phase int

const (
	phaseDate phase = iota
	phaseStore
	phaseProduct
	phaseAnotherStore
)

type interactiveModel struct {
	phase        phase
	textInput    textinput.Model
	date         string
	dateOffset   int
	currentStore string
	entries      []models.StoreEntry
	currentItems []models.ProductPrice
	productNames []string
	storeNames   []string
	warnings     []string
	done         bool
	cancelled    bool
}

var (
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	itemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	storeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	priceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	totalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

func dateHint(dateStr string) string {
	now := time.Now()
	loc := now.Location()
	t, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return ""
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	diff := int(today.Sub(t).Hours() / 24)

	dayName := t.Format("Monday")

	switch diff {
	case 0:
		return fmt.Sprintf("Today, %s", dayName)
	case 1:
		return fmt.Sprintf("Yesterday, %s", dayName)
	case -1:
		return fmt.Sprintf("Tomorrow, %s", dayName)
	default:
		if diff > 1 && diff <= 7 {
			return fmt.Sprintf("%d days ago, %s", diff, dayName)
		}
		return dayName
	}
}

func truncateMatches(matches []string, maxLen int) string {
	result := strings.Join(matches, ", ")
	if len(result) <= maxLen {
		return result
	}
	var parts []string
	length := 0
	for _, m := range matches {
		added := len(m)
		if len(parts) > 0 {
			added += 2
		}
		if length+added+3 > maxLen {
			break
		}
		parts = append(parts, m)
		length += added
	}
	if len(parts) == 0 {
		return result[:maxLen-3] + "..."
	}
	return strings.Join(parts, ", ") + ", ..."
}

func newInteractiveModel() interactiveModel {
	ti := textinput.New()
	ti.Focus()
	ti.ShowSuggestions = true
	ti.KeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("tab"))
	ti.KeyMap.NextSuggestion = key.NewBinding(key.WithKeys("down"))
	ti.KeyMap.PrevSuggestion = key.NewBinding(key.WithKeys("up"))

	defaultDate := models.Today()

	m := interactiveModel{
		phase:        phaseDate,
		textInput:    ti,
		productNames: storage.ProductNames(),
		storeNames:   storage.UniqueStores(),
	}

	hint := dateHint(defaultDate)
	m.textInput.SetValue(defaultDate)
	m.textInput.Placeholder = hint
	m.textInput.Prompt = labelStyle.Render("Date: ")
	m.textInput.CursorEnd()

	return m
}

func (m interactiveModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit

		case "up", "down":
			if m.phase == phaseDate {
				if msg.String() == "up" {
					m.dateOffset++
				} else {
					m.dateOffset--
				}
				d := time.Now().AddDate(0, 0, m.dateOffset).Format("2006-01-02")
				hint := dateHint(d)
				m.textInput.SetValue(d)
				m.textInput.Placeholder = hint
				m.textInput.CursorEnd()
				return m, nil
			}

		case "enter":
			value := strings.TrimSpace(m.textInput.Value())
			m.warnings = nil

			switch m.phase {
			case phaseDate:
				if value == "" {
					value = models.Today()
				}
				if !dateRegex.MatchString(value) {
					m.warnings = append(m.warnings, "Invalid date format (YYYY-MM-DD)")
					return m, nil
				}
				m.date = value
				m.phase = phaseStore
				m.textInput.Reset()
				m.textInput.Prompt = labelStyle.Render("Store: ")
				m.textInput.Placeholder = strings.Join(m.storeNames, ", ")
				m.textInput.SetSuggestions(m.storeNames)
				return m, nil

			case phaseStore:
				if value == "" {
					m.warnings = append(m.warnings, "Store is required")
					return m, nil
				}
				m.currentStore = strings.ToLower(value)
				m.phase = phaseProduct
				m.currentItems = nil
				m.textInput.Reset()
				m.textInput.Placeholder = "product price (empty to finish)"
				m.textInput.Prompt = labelStyle.Render("> ")
				m.textInput.SetSuggestions(m.productNames)
				return m, nil

			case phaseProduct:
				if value == "" {
					if len(m.currentItems) > 0 {
						m.entries = append(m.entries, models.StoreEntry{
							Store:    m.currentStore,
							Date:     m.date,
							Products: m.currentItems,
						})
					}
					m.currentItems = nil
					m.currentStore = ""
					m.phase = phaseAnotherStore
					m.textInput.Reset()
					m.textInput.Placeholder = ""
					m.textInput.Prompt = labelStyle.Render("Add another store? [y/N]: ")
					m.textInput.SetSuggestions(nil)
					return m, nil
				}

				parts := strings.Fields(value)
				if len(parts) < 2 {
					m.warnings = append(m.warnings, "Format: product-name price (e.g. ground-beef 200)")
					m.textInput.Reset()
					return m, nil
				}
				productName := strings.ToLower(parts[0])
				price, err := strconv.ParseFloat(parts[1], 64)
				if err != nil {
					m.warnings = append(m.warnings, fmt.Sprintf("Invalid price: %s", parts[1]))
					m.textInput.Reset()
					return m, nil
				}

				m.currentItems = append(m.currentItems, models.ProductPrice{
					Product: productName,
					Price:   price,
				})

				found := false
				for _, n := range m.productNames {
					if n == productName {
						found = true
						break
					}
				}
				if !found {
					m.productNames = append(m.productNames, productName)
					m.textInput.SetSuggestions(m.productNames)
				}

				m.textInput.Reset()
				return m, nil

			case phaseAnotherStore:
				answer := strings.ToLower(value)
				if answer == "y" || answer == "yes" {
					m.phase = phaseStore
					m.textInput.Reset()
					m.textInput.Prompt = labelStyle.Render("Store: ")
					m.textInput.Placeholder = strings.Join(m.storeNames, ", ")
					m.textInput.SetSuggestions(m.storeNames)
					return m, nil
				}
				m.done = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m interactiveModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Header: always show date once set
	if m.date != "" {
		hint := dateHint(m.date)
		b.WriteString(headerStyle.Render("─── kvit ───") + "\n")
		b.WriteString(labelStyle.Render("  Date:  ") + m.date + "  " + hintStyle.Render("("+hint+")") + "\n")
	}

	// Show completed store entries
	for _, entry := range m.entries {
		b.WriteString(storeStyle.Render(fmt.Sprintf("  Store: %s", entry.Store)) + "\n")
		for _, p := range entry.Products {
			b.WriteString(itemStyle.Render(fmt.Sprintf("    + %-20s", p.Product)) +
				priceStyle.Render(fmt.Sprintf("%10.2f %s", p.Price, config.Currency())) + "\n")
		}
		b.WriteString("\n")
	}

	// Show current store and items being added
	if m.currentStore != "" && m.phase == phaseProduct {
		b.WriteString(storeStyle.Render(fmt.Sprintf("  Store: %s", m.currentStore)) + "\n")
		for _, p := range m.currentItems {
			b.WriteString(itemStyle.Render(fmt.Sprintf("    + %-20s", p.Product)) +
				priceStyle.Render(fmt.Sprintf("%10.2f %s", p.Price, config.Currency())) + "\n")
		}
	}

	// Warnings
	for _, w := range m.warnings {
		b.WriteString(warnStyle.Render("  ⚠ "+w) + "\n")
	}

	b.WriteString("\n" + m.textInput.View())

	// Inline hints after the text input on the same line
	switch m.phase {
	case phaseDate:
		typed := strings.TrimSpace(m.textInput.Value())
		if dateRegex.MatchString(typed) {
			hint := dateHint(typed)
			if hint != "" {
				b.WriteString(" " + hintStyle.Render("("+hint+")"))
			}
		}
	case phaseStore:
		typed := strings.TrimSpace(m.textInput.Value())
		if typed != "" {
			var matches []string
			lower := strings.ToLower(typed)
			for _, s := range m.storeNames {
				if strings.HasPrefix(strings.ToLower(s), lower) {
					matches = append(matches, s)
				}
			}
			if len(matches) > 0 && !(len(matches) == 1 && strings.ToLower(matches[0]) == lower) {
				b.WriteString(" " + hintStyle.Render("("+truncateMatches(matches, 40)+")"))
			}
		}
	case phaseProduct:
		typed := strings.Fields(strings.TrimSpace(m.textInput.Value()))
		if len(typed) == 1 {
			var matches []string
			lower := strings.ToLower(typed[0])
			for _, p := range m.productNames {
				if strings.HasPrefix(strings.ToLower(p), lower) {
					matches = append(matches, p)
				}
			}
			if len(matches) > 0 && !(len(matches) == 1 && strings.ToLower(matches[0]) == lower) {
				b.WriteString(" " + hintStyle.Render("("+truncateMatches(matches, 40)+")"))
			}
		}
	}

	return b.String()
}

func runInteractive(cmd *cobra.Command, args []string) {
	m := newInteractiveModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result := finalModel.(interactiveModel)

	if result.cancelled || len(result.entries) == 0 {
		fmt.Println("Nothing to save.")
		return
	}

	if ConfirmAndSave(result.entries) {
		HandleUnknownProducts(result.entries)
	}
}
