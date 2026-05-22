package bidget

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type menuState int

const (
	stateMenu menuState = iota
	stateSettings
)

type Model struct {
	state      menuState
	cursor     int
	env        map[string]string
	inputKey   textinput.Model
	inputValue textinput.Model
}

func NewMenuModel(env map[string]string) Model {
	keyInput := textinput.New()
	keyInput.Placeholder = "API_KEY"
	keyInput.Focus()

	valueInput := textinput.New()
	valueInput.Placeholder = "Enter value"

	return Model{
		state:      stateMenu,
		cursor:     0,
		env:        env,
		inputKey:   keyInput,
		inputValue: valueInput,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			switch msg.String() {
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < 1 {
					m.cursor++
				}
			case "enter":
				if m.cursor == 0 {
					// Start bot logic here
				} else {
					m.state = stateSettings
				}
			case "q":
				return m, tea.Quit
			}
		case stateSettings:
			switch msg.String() {
			case "esc":
				m.state = stateMenu
			case "enter":
				key := m.inputKey.Value()
				val := m.inputValue.Value()
				m.env[key] = val
				// Save to .env if needed
			}
			var cmd tea.Cmd
			m.inputKey, cmd = m.inputKey.Update(msg)
			m.inputValue, _ = m.inputValue.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateMenu:
		options := []string{"▶️ Start", "⚙️ Settings"}
		var b strings.Builder
		b.WriteString("🟢 Bitget Bot Menu\n\n")
		for i, opt := range options {
			cursor := "  "
			if i == m.cursor {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, opt))
		}
		b.WriteString("\nUse ↑ ↓ to navigate, Enter to select.")
		return b.String()
	case stateSettings:
		return fmt.Sprintf(
			"🔧 Settings\n\nKey: %s\nValue: %s\n\nPress Enter to save, Esc to cancel.",
			m.inputKey.View(), m.inputValue.View(),
		)
	}
	return ""
}
