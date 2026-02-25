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
	} else if m.copyModal.visible {
		modal = m.copyModalView()
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	} else if m.moveModal.visible {
		modal = m.moveModalView()
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	} else if m.deleteModal.visible {
		modal = m.deleteModalView()
		paneHeight = max(1, paneHeight-lipgloss.Height(modal)-1)
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	leftPaneView := renderPane(leftWidth, paneHeight, m.leftPane.table.View())
	rightPaneView := renderPane(rightWidth, paneHeight, m.rightPane.table.View())

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftPaneView, rightPaneView)
	background := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, tables)
	if !m.searchModalVisible && !m.pickerModalVisible && !m.copyModal.visible && !m.moveModal.visible && !m.deleteModal.visible {
		return background
	}

	return overlayCentered(background, modal, m.width, m.height)
}

func (m Model) copyModalView() string {
	if m.copyModal.progress.inProgress {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.header).
			Render("Copying")

		frame := deleteSpinnerFrames[m.copyModal.progress.frame%len(deleteSpinnerFrames)]
		progress := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("%s Processing %d/%d item(s)", frame, m.copyModal.progress.done+1, max(1, m.copyModal.progress.total)))

		current := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("Current: %q", m.copyModal.progress.current))

		hint := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render("Copy operation is in progress...")

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.border).
			Padding(1, 2).
			Width(84).
			Render(lipgloss.JoinVertical(lipgloss.Left, title, progress, current, hint))
	}

	if m.copyModal.hasResult || m.copyModal.planningErr != nil {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.header).
			Render("Copy Result")

		var summary string
		if m.copyModal.planningErr != nil {
			summary = fmt.Sprintf("Failed to start copy: %v", m.copyModal.planningErr)
		} else {
			summary = fmt.Sprintf(
				"Planned: %d  Copied: %d  Skipped: %d  Failed: %d",
				m.copyModal.planned,
				len(m.copyModal.result.Copied),
				len(m.copyModal.result.Skipped),
				len(m.copyModal.result.Failed),
			)
		}
		summaryView := lipgloss.NewStyle().Foreground(m.theme.text).Render(summary)

		hint := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render("enter/esc: close")

		lines := []string{title, summaryView}
		if len(m.copyModal.result.Failed) > 0 {
			first := m.copyModal.result.Failed[0]
			lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.text).Render(
				fmt.Sprintf("First error (%s): %v", first.Stage, first.Err),
			))
		}
		lines = append(lines, hint)

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.border).
			Padding(1, 2).
			Width(72).
			Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.header).
		Render("Confirm Copy")

	targetPath := m.leftPane.path
	if m.copyModal.destinationPane == paneRight {
		targetPath = m.rightPane.path
	}
	question := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render(fmt.Sprintf("Copy %d item(s) to %s?", len(m.copyModal.entries), targetPath))

	policy := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("Conflict policy: skip existing")

	hint := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("y: start  n/esc: cancel")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.border).
		Padding(1, 2).
		Width(72).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, question, policy, hint))
}

func (m Model) moveModalView() string {
	if m.moveModal.progress.inProgress {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.header).
			Render("Moving")

		frame := deleteSpinnerFrames[m.moveModal.progress.frame%len(deleteSpinnerFrames)]
		progress := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("%s Processing %d/%d item(s)", frame, m.moveModal.progress.done+1, max(1, m.moveModal.progress.total)))

		current := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("Current: %q", m.moveModal.progress.current))

		hint := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render("Move operation is in progress...")

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.border).
			Padding(1, 2).
			Width(84).
			Render(lipgloss.JoinVertical(lipgloss.Left, title, progress, current, hint))
	}

	if m.moveModal.hasResult || m.moveModal.planningErr != nil {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.header).
			Render("Move Result")

		var summary string
		if m.moveModal.planningErr != nil {
			summary = fmt.Sprintf("Failed to start move: %v", m.moveModal.planningErr)
		} else {
			summary = fmt.Sprintf(
				"Planned: %d  Moved: %d  Skipped: %d  Failed: %d",
				m.moveModal.planned,
				len(m.moveModal.result.Copied),
				len(m.moveModal.result.Skipped),
				len(m.moveModal.result.Failed),
			)
		}
		summaryView := lipgloss.NewStyle().Foreground(m.theme.text).Render(summary)

		hint := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render("enter/esc: close")

		lines := []string{title, summaryView}
		if len(m.moveModal.result.Failed) > 0 {
			first := m.moveModal.result.Failed[0]
			lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.text).Render(
				fmt.Sprintf("First error (%s): %v", first.Stage, first.Err),
			))
		}
		lines = append(lines, hint)

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.border).
			Padding(1, 2).
			Width(72).
			Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.header).
		Render("Confirm Move")

	targetPath := m.leftPane.path
	if m.moveModal.destinationPane == paneRight {
		targetPath = m.rightPane.path
	}
	question := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render(fmt.Sprintf("Move %d item(s) to %s?", len(m.moveModal.entries), targetPath))

	warning := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("Source items will be deleted after successful copy.")

	policy := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("Conflict policy: skip existing")

	hint := lipgloss.NewStyle().
		Foreground(m.theme.text).
		Render("y: start  n/esc: cancel")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.border).
		Padding(1, 2).
		Width(72).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, question, warning, policy, hint))
}

func (m Model) deleteModalView() string {
	if m.deleteModal.progress.inProgress {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.header).
			Render("Deleting")

		frame := deleteSpinnerFrames[m.deleteModal.progress.frame%len(deleteSpinnerFrames)]
		progress := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("%s Processing %d/%d item(s)", frame, m.deleteModal.progress.done+1, m.deleteModal.progress.total))

		target := lipgloss.NewStyle().
			Foreground(m.theme.text).
			Render(fmt.Sprintf("Current: %q", m.deleteModal.progress.current))

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
	switch len(m.deleteModal.entries) {
	case 0:
		targetLabel = "Delete selected item(s)?"
	case 1:
		target := m.deleteModal.entries[0]
		if target.Kind == KindDirectory {
			targetLabel = fmt.Sprintf("Delete directory %q recursively?", target.Name)
		} else {
			targetLabel = fmt.Sprintf("Delete %q?", target.Name)
		}
	default:
		var dirCount int
		var fileCount int
		for _, entry := range m.deleteModal.entries {
			if entry.Kind == KindDirectory {
				dirCount++
				continue
			}
			fileCount++
		}
		targetLabel = fmt.Sprintf("Delete %d item(s) (%d directorie(s), %d file(s))?", len(m.deleteModal.entries), dirCount, fileCount)
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
