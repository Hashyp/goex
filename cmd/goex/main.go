package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"defaultdevcontainer/internal/app"
)

func main() {
	p := tea.NewProgram(app.NewModel(), tea.WithAltScreen())

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}
