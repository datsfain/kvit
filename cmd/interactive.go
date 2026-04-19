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
	editMode      bool  // cursor navigation through entries
	editCursor    int   // index into editableLines()
	editing       bool  // actively editing a field via text input
	editPrevPhase phase // phase to restore on edit exit
}

// editLine points to a single editable row.
// field is one of: "store", "date" (for header rows) or "product" (for item rows).
type editLine struct {
	entryIdx   int
	productIdx int    // -1 for store/date rows
	field      string // "store" | "date" | "product"
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

func (m interactiveModel) editableLines() []editLine {
	var lines []editLine
	for i, entry := range m.entries {
		lines = append(lines, editLine{entryIdx: i, productIdx: -1, field: "store"})
		lines = append(lines, editLine{entryIdx: i, productIdx: -1, field: "date"})
		for j := range entry.Products {
			lines = append(lines, editLine{entryIdx: i, productIdx: j, field: "product"})
		}
	}
	return lines
}

// flushCurrentItems commits any in-progress items to entries.
// Called before entering edit mode so there's a single source of truth.
func (m *interactiveModel) flushCurrentItems() {
	if len(m.currentItems) == 0 {
		return
	}
	m.entries = append(m.entries, models.StoreEntry{
		Store:    m.currentStore,
		Date:     m.date,
		Products: m.currentItems,
	})
	m.currentItems = nil
	m.currentStore = ""
}

func (m *interactiveModel) enterEditMode() {
	m.editPrevPhase = m.phase
	m.flushCurrentItems()
	lines := m.editableLines()
	if len(lines) == 0 {
		m.warnings = append(m.warnings, "Nothing to edit yet")
		return
	}
	m.editMode = true
	m.editCursor = len(lines) - 1
	m.editing = false
	m.warnings = nil
}

func (m *interactiveModel) exitEditMode() {
	m.editMode = false
	m.editing = false
	m.textInput.Reset()
	m.textInput.Placeholder = ""

	// If the user was adding products, lift the last entry back into
	// currentItems so they can keep adding to the same store/date.
	if m.editPrevPhase == phaseProduct && len(m.entries) > 0 {
		last := m.entries[len(m.entries)-1]
		m.entries = m.entries[:len(m.entries)-1]
		m.currentStore = last.Store
		m.date = last.Date
		m.currentItems = last.Products
		m.phase = phaseProduct
		m.textInput.Prompt = labelStyle.Render("> ")
		m.textInput.Placeholder = "product price (empty to finish)"
		m.textInput.SetSuggestions(m.productNames)
		return
	}

	m.phase = phaseAnotherStore
	m.textInput.Prompt = labelStyle.Render("Add another store? [y/N]: ")
	m.textInput.SetSuggestions(nil)
}

func (m *interactiveModel) editStartField() {
	lines := m.editableLines()
	if m.editCursor >= len(lines) {
		return
	}
	line := lines[m.editCursor]
	entry := m.entries[line.entryIdx]

	var prefill, prompt string
	switch line.field {
	case "store":
		prefill = entry.Store
		prompt = "Edit store: "
	case "date":
		prefill = entry.Date
		prompt = "Edit date: "
	case "product":
		p := entry.Products[line.productIdx]
		prefill = p.Product + " " + strconv.FormatFloat(p.Price, 'f', -1, 64)
		prompt = "Edit item: "
	}

	m.editing = true
	m.textInput.Reset()
	m.textInput.SetValue(prefill)
	m.textInput.Prompt = labelStyle.Render(prompt)
	m.textInput.Placeholder = ""
	m.textInput.SetSuggestions(nil)
	m.textInput.CursorEnd()
}

func (m *interactiveModel) editApply(value string) bool {
	lines := m.editableLines()
	if m.editCursor >= len(lines) {
		return false
	}
	line := lines[m.editCursor]

	switch line.field {
	case "store":
		name := strings.TrimSpace(value)
		if name == "" {
			m.warnings = append(m.warnings, "Store name required")
			return false
		}
		m.entries[line.entryIdx].Store = strings.ToLower(name)
	case "date":
		d := strings.TrimSpace(value)
		if !dateRegex.MatchString(d) {
			m.warnings = append(m.warnings, "Invalid date (YYYY-MM-DD)")
			return false
		}
		m.entries[line.entryIdx].Date = d
	case "product":
		parts := strings.Fields(strings.TrimSpace(value))
		if len(parts) < 2 {
			m.warnings = append(m.warnings, "Format: product price")
			return false
		}
		price, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			m.warnings = append(m.warnings, "Invalid price: "+parts[1])
			return false
		}
		m.entries[line.entryIdx].Products[line.productIdx].Product = strings.ToLower(parts[0])
		m.entries[line.entryIdx].Products[line.productIdx].Price = price
	}
	return true
}

func (m *interactiveModel) editDelete() {
	lines := m.editableLines()
	if m.editCursor >= len(lines) {
		return
	}
	line := lines[m.editCursor]

	if line.productIdx == -1 {
		// Remove the whole store entry
		m.entries = append(m.entries[:line.entryIdx], m.entries[line.entryIdx+1:]...)
	} else {
		// Remove one product; if store becomes empty, remove the store too
		entry := &m.entries[line.entryIdx]
		entry.Products = append(entry.Products[:line.productIdx], entry.Products[line.productIdx+1:]...)
		if len(entry.Products) == 0 {
			m.entries = append(m.entries[:line.entryIdx], m.entries[line.entryIdx+1:]...)
		}
	}

	// Clamp cursor to new length
	newLen := len(m.editableLines())
	if newLen == 0 {
		m.exitEditMode()
		return
	}
	if m.editCursor >= newLen {
		m.editCursor = newLen - 1
	}
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
		// Edit mode: intercept all keys before normal phase handling
		if m.editMode {
			return m.updateEditMode(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			m.done = true
			return m, tea.Quit

		case "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit

		case "ctrl+e":
			m.enterEditMode()
			return m, nil

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

func (m interactiveModel) updateEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Actively typing a new value in the text input
	if m.editing {
		switch msg.String() {
		case "esc":
			m.editing = false
			m.textInput.Reset()
			return m, nil
		case "enter":
			value := strings.TrimSpace(m.textInput.Value())
			if m.editApply(value) {
				m.editing = false
				m.textInput.Reset()
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Navigation mode
	switch msg.String() {
	case "ctrl+c":
		m.cancelled = true
		m.done = true
		return m, tea.Quit
	case "esc":
		m.exitEditMode()
		return m, nil
	case "up", "k":
		if m.editCursor > 0 {
			m.editCursor--
		}
		return m, nil
	case "down", "j":
		if m.editCursor < len(m.editableLines())-1 {
			m.editCursor++
		}
		return m, nil
	case "enter":
		m.editStartField()
		return m, nil
	case "d":
		m.editDelete()
		return m, nil
	}
	return m, nil
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

	// Show completed store entries (with cursor in edit mode)
	lineIdx := 0
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	renderCursor := func(indent string) string {
		if m.editMode && m.editCursor == lineIdx {
			return indent + cursorStyle.Render("▸ ")
		}
		return indent + "  "
	}
	for _, entry := range m.entries {
		b.WriteString(renderCursor("") + storeStyle.Render(fmt.Sprintf("Store: %s", entry.Store)) + "\n")
		lineIdx++
		b.WriteString(renderCursor("") + labelStyle.Render(fmt.Sprintf("Date:  %s", entry.Date)) + "\n")
		lineIdx++
		for _, p := range entry.Products {
			b.WriteString(renderCursor("  ") + itemStyle.Render(fmt.Sprintf("+ %-20s", p.Product)) +
				priceStyle.Render(fmt.Sprintf("%10.2f %s", p.Price, config.Currency())) + "\n")
			lineIdx++
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

	// Edit mode UI
	if m.editMode {
		if m.editing {
			b.WriteString("\n" + m.textInput.View())
			b.WriteString("\n\n" + hintStyle.Render("  enter: save · esc: cancel"))
		} else {
			b.WriteString("\n" + hintStyle.Render("  ── edit mode ──"))
			b.WriteString("\n" + hintStyle.Render("  ↑/↓ navigate · enter: edit · d: delete · esc: exit"))
		}
		return b.String()
	}

	b.WriteString("\n" + m.textInput.View())

	// Persistent hint about edit mode availability
	if len(m.entries) > 0 || len(m.currentItems) > 0 {
		b.WriteString("\n\n" + hintStyle.Render("  ctrl+e: edit previous entries"))
	}

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
