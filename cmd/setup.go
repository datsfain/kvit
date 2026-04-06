package cmd

import (
	"fmt"
	"kvit/config"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type setupPhase int

const (
	setupCurrency setupPhase = iota
	setupLanguages
)

type setupModel struct {
	phase     setupPhase
	textInput textinput.Model
	currency  string
	languages []string
	done      bool
	cancelled bool
}

func newSetupModel() setupModel {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = labelStyle.Render("Currency code: ")
	ti.Placeholder = "e.g. USD, EUR, DKK, GBP"
	ti.CharLimit = 10

	return setupModel{
		phase:     setupCurrency,
		textInput: ti,
	}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			value := strings.TrimSpace(m.textInput.Value())

			switch m.phase {
			case setupCurrency:
				if value == "" {
					return m, nil
				}
				m.currency = strings.ToUpper(value)
				m.phase = setupLanguages
				m.textInput.Reset()
				m.textInput.Prompt = labelStyle.Render("Receipt languages: ")
				m.textInput.Placeholder = "e.g. English, Danish, German"
				m.textInput.CharLimit = 100
				return m, nil

			case setupLanguages:
				if value == "" {
					return m, nil
				}
				for _, lang := range strings.Split(value, ",") {
					lang = strings.TrimSpace(lang)
					if lang != "" {
						m.languages = append(m.languages, lang)
					}
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

func (m setupModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n" + headerStyle.Render("─── kvit setup ───") + "\n\n")

	if m.phase == setupCurrency {
		b.WriteString(hintStyle.Render("  What currency do you use? (3-letter code)") + "\n\n")
	}

	if m.currency != "" {
		b.WriteString(successStyle.Render("  ✓ ") + "Currency: " + successStyle.Render(m.currency) + "\n\n")
	}

	if m.phase == setupLanguages {
		b.WriteString(hintStyle.Render("  What language(s) are your receipts in? (comma-separated)") + "\n\n")
	}

	b.WriteString("  " + m.textInput.View())

	return b.String()
}

// RunSetup runs the first-time setup TUI and saves the config.
// Returns true if setup was completed successfully.
func RunSetup() bool {
	fmt.Println(headerStyle.Render("Welcome to kvit!") + " Let's set up a few things.\n")

	m := newSetupModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return false
	}

	result := finalModel.(setupModel)
	if result.cancelled || result.currency == "" {
		fmt.Println("Setup cancelled.")
		return false
	}

	c := config.Load()
	c.Currency = result.currency
	c.Languages = result.languages

	if err := config.Save(c); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return false
	}

	fmt.Println()
	fmt.Println(successStyle.Render("✓ Setup complete!"))
	fmt.Printf("  Currency:  %s\n", result.currency)
	fmt.Printf("  Languages: %s\n", strings.Join(result.languages, ", "))
	fmt.Println()

	return true
}
