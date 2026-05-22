package bidget

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q, esc or ctrl+c", "to exit"),
	),
}

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle  = helpStyle.UnsetMargins()
	appStyle  = lipgloss.NewStyle().Margin(1, 2, 0, 2)
)

type keyMap struct {
	Quit key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{},       // first column
		{k.Quit}, // second column
	}
}

func (r balanceMsg) String() string {
	if r.date.IsZero() {
		return dotStyle.Render(strings.Repeat(".", 30))
	}

	if len(r.balances) == 0 {
		return r.date.Format("2006-01-02 15:04:05")
	}

	return fmt.Sprintf("%s \n%s", r.date.Format("2006-01-02 15:04:05"), list.New(r.balances))
}

type model struct {
	keys     keyMap
	help     help.Model
	timer    timer.Model
	spinner  spinner.Model
	balances []balanceMsg
	err      error
}

func InitialModel() model {
	const numLastResults = 5

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		keys:     keys,
		timer:    timer.NewWithInterval(2*time.Second, time.Millisecond),
		help:     help.New(),
		spinner:  s,
		balances: make([]balanceMsg, numLastResults),
	}
}

type balanceMsg struct {
	balances []string
	date     time.Time
	err      error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.timer.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate
		// its view as needed.
		m.help.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		return m, tea.Batch(CheckBalanceCmd())
	case balanceMsg:
		m.balances = append(m.balances[1:], msg)
		m.err = msg.err
		m.timer = timer.NewWithInterval(timeout, time.Millisecond)
		return m, m.timer.Init()
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	s := "🟢 Bitget Auto-Sell Bot\n"
	if m.err != nil {
		s += "❌ Error: " + m.err.Error() + "\n"
	}

	balanceItems := []any{}
	for _, b := range m.balances {
		balanceItems = append(balanceItems, b)
	}

	s += fmt.Sprintf("\n %s", list.New("Balance", list.New(balanceItems...)))
	s += fmt.Sprintf("\n\n%s%s\n", m.spinner.View(), m.timer.View())

	s += m.help.View(m.keys)
	return appStyle.Render(s)
}
