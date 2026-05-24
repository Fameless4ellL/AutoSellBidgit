package main

import (
	"autosell/internal/bidget"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	bidget.LoadEnvInteractive()

	mod := bidget.InitialModel()
	p := tea.NewProgram(mod)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
