package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) activePaneRef() *Pane {
	if m.activePane == paneRight {
		return &m.rightPane
	}

	return &m.leftPane
}

func (m *Model) setActivePane(pane activePane) {
	m.activePane = pane
	m.leftPane.table = m.leftPane.table.Focused(pane == paneLeft)
	m.rightPane.table = m.rightPane.table.Focused(pane == paneRight)
}

func (m *Model) toggleHeader() {
	m.leftPane.table = m.leftPane.table.WithHeaderVisibility(!m.leftPane.table.GetHeaderVisibility())
	m.rightPane.table = m.rightPane.table.WithHeaderVisibility(!m.rightPane.table.GetHeaderVisibility())
}

func (m *Model) applyTheme() {
	m.leftPane.table = applyThemeToTable(m.leftPane.table, m.theme)
	m.rightPane.table = applyThemeToTable(m.rightPane.table, m.theme)
}

func (m *Model) toggleTheme() {
	m.theme = nextTheme(m.theme.name)
	m.applyTheme()
}

func (m *Model) updateAllTables(msg tea.Msg) []tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2)

	left, leftCmd := m.leftPane.table.Update(msg)
	m.leftPane.table = left
	cmds = append(cmds, leftCmd)

	right, rightCmd := m.rightPane.table.Update(msg)
	m.rightPane.table = right
	cmds = append(cmds, rightCmd)

	return cmds
}

func (m *Model) updateActiveTable(msg tea.Msg) []tea.Cmd {
	cmds := []tea.Cmd{}
	active := m.activePaneRef()
	updated, cmd := active.table.Update(msg)
	active.table = updated
	cmds = append(cmds, cmd)
	return cmds
}

func (m *Model) jumpToLastRow() {
	active := m.activePaneRef()
	lastIndex := len(active.table.GetVisibleRows()) - 1
	if lastIndex < 0 {
		return
	}

	active.table = active.table.WithHighlightedRow(lastIndex)
}

func (m *Model) handleKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "q":
		return true, []tea.Cmd{tea.Quit}
	case "tab", "shift+tab":
		if m.activePane == paneLeft {
			m.setActivePane(paneRight)
		} else {
			m.setActivePane(paneLeft)
		}
		return true, nil
	case "i":
		m.toggleHeader()
		return true, nil
	case "t":
		m.toggleTheme()
		m.status = fmt.Sprintf("Theme: %s", m.theme.name)
		return true, nil
	case "G", "shift+g", "end":
		m.jumpToLastRow()
		return true, nil
	case "enter", "l":
		if err := m.activePaneRef().enterHighlightedDirectory(m.fs); err != nil {
			m.status = err.Error()
		} else {
			m.status = ""
		}
		return true, nil
	case "backspace", "h":
		if err := m.activePaneRef().goParent(m.fs); err != nil {
			m.status = err.Error()
		} else {
			m.status = ""
		}
		return true, nil
	default:
		return false, nil
	}
}
