package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var deleteSpinnerFrames = []string{"|", "/", "-", "\\"}

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
	if m.pickerModalVisible {
		modal = m.panePickerModalView()
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	} else if m.searchModalVisible {
		modal = m.searchModalView()
		// Keep tables visible while modal is open.
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	} else if m.deleteModalVisible {
		modal = m.deleteModalView()
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	leftPaneView := renderPane(leftWidth, paneHeight, m.leftPane.table.View())
	rightPaneView := renderPane(rightWidth, paneHeight, m.rightPane.table.View())

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftPaneView, rightPaneView)
	background := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, tables)
	if !m.searchModalVisible && !m.pickerModalVisible && !m.deleteModalVisible {
		return background
	}

	return overlayCentered(background, modal, m.width, m.height)
}

func (m Model) deleteModalView() string {
	if m.deleteInProgress {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.header).
			Render("Deleting")

		frame := deleteSpinnerFrames[m.deleteProgressFrame%len(deleteSpinnerFrames)]
		progress := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("%s Processing %d/%d item(s)", frame, m.deleteProgressDone+1, m.deleteProgressTotal))

		target := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("Current: %q", m.deleteProgressName))

		hint := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render("Deletion is in progress...")

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.border).
			Padding(1, 2).
			Width(56).
			Render(lipgloss.JoinVertical(lipgloss.Left, title, progress, target, hint))
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.header).
		Render("Confirm Delete")

	targetLabel := ""
	switch len(m.deleteTargetEntries) {
	case 0:
		targetLabel = "Delete selected item(s)?"
	case 1:
		target := m.deleteTargetEntries[0]
		if target.Kind == KindDirectory {
			targetLabel = fmt.Sprintf("Delete directory %q recursively?", target.Name)
		} else {
			targetLabel = fmt.Sprintf("Delete %q?", target.Name)
		}
	default:
		var dirCount int
		var fileCount int
		for _, entry := range m.deleteTargetEntries {
			if entry.Kind == KindDirectory {
				dirCount++
				continue
			}
			fileCount++
		}
		targetLabel = fmt.Sprintf("Delete %d item(s) (%d directorie(s), %d file(s))?", len(m.deleteTargetEntries), dirCount, fileCount)
	}

	question := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render(targetLabel)

	warning := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("Directories are deleted recursively. This cannot be undone.")

	hint := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("y: yes  n/esc: cancel")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.border).
		Padding(1, 2).
		Width(56).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, question, warning, hint))
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

func (m Model) panePickerModalView() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.header).
		Render(fmt.Sprintf("Select Backend (%s pane)", paneName(m.pickerTargetPane)))

	rows := make([]string, 0, len(paneBackendChoices))
	for index, choice := range paneBackendChoices {
		label := paneBackendLabel(choice)
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(m.theme.text)
		if index == m.pickerChoiceIndex {
			prefix = "> "
			style = style.
				Foreground(m.theme.highlightFG).
				Background(m.theme.highlightBG).
				Bold(true)
		}
		rows = append(rows, style.Render(prefix+label))
	}

	hint := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("up/down or j/k: move  enter: accept  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, rows...)...)
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.border).
		Padding(1, 2).
		Width(56).
		Render(lipgloss.JoinVertical(lipgloss.Left, content, hint))
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
