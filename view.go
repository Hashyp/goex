package main

import "github.com/charmbracelet/lipgloss"

func renderPane(width, height int, content string) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Left, lipgloss.Top).
		Render(content)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	paneHeight := m.paneHeight()
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	leftPaneView := renderPane(leftWidth, paneHeight, m.leftPane.table.View())
	rightPaneView := renderPane(rightWidth, paneHeight, m.rightPane.table.View())

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftPaneView, rightPaneView)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, tables)
}
