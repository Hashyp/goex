package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"defaultdevcontainer/internal/app"
)

func main() {
	p := tea.NewProgram(app.NewModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
