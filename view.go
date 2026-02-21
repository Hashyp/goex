package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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
	var modal string
	if m.searchModalVisible {
		modal = m.searchModalView()
		// Keep tables visible while modal is open.
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	leftPaneView := renderPane(leftWidth, paneHeight, m.leftPane.table.View())
	rightPaneView := renderPane(rightWidth, paneHeight, m.rightPane.table.View())

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftPaneView, rightPaneView)
	background := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, tables)
	if !m.searchModalVisible {
		return background
	}

	return overlayCentered(background, modal, m.width, m.height)
}

func (m Model) searchModalView() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.header).
		Render("Search (Regular Expression)")

	hint := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("enter: accept  esc: cancel")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.border).
		Padding(1, 2).
		Width(60).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, m.searchInput.View(), hint))
}

func overlayCentered(background, overlay string, width, height int) string {
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayWidth := lipgloss.Width(overlay)
	startX := max(0, (width-overlayWidth)/2)
	startY := max(0, (height-len(overlayLines))/2)
	endX := min(width, startX+overlayWidth)

	for i, line := range overlayLines {
		y := startY + i
		if y < 0 || y >= len(bgLines) {
			continue
		}

		left := ansi.Cut(bgLines[y], 0, startX)
		right := ansi.Cut(bgLines[y], endX, width)
		bgLines[y] = left + line + right
	}

	return strings.Join(bgLines, "\n")
}
