package main

import "github.com/charmbracelet/lipgloss"

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	paneHeight := m.paneHeight()
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	leftPaneView := lipgloss.NewStyle().
		Width(leftWidth).
		Height(paneHeight).
		Align(lipgloss.Left, lipgloss.Top).
		Render(m.leftPane.table.View())

	rightPaneView := lipgloss.NewStyle().
		Width(rightWidth).
		Height(paneHeight).
		Align(lipgloss.Left, lipgloss.Top).
		Render(m.rightPane.table.View())

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftPaneView, rightPaneView)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, tables)
}
