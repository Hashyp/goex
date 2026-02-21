package app

import (
	"fmt"
	"regexp"

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
	m.leftPane.table = applyThemeToTable(m.leftPane.table, m.theme, m.leftPane.selected)
	m.rightPane.table = applyThemeToTable(m.rightPane.table, m.theme, m.rightPane.selected)
}

func (m *Model) toggleTheme() {
	m.themeIndex = nextThemeIndex(m.themeIndex)
	m.theme = themes[m.themeIndex]
	m.applyTheme()
	m.leftPane.refreshRows(m.theme)
	m.rightPane.refreshRows(m.theme)
}

func (m *Model) toggleHiddenFiles() {
	active := m.activePaneRef()
	active.showHidden = !active.showHidden
	if err := active.reload(m.fs, m.theme); err != nil {
		m.status = err.Error()
		return
	}
	if active.showHidden {
		m.status = "Hidden files: shown"
		return
	}

	m.status = "Hidden files: hidden"
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

func (m *Model) selectHighlightedAndAdvance() {
	active := m.activePaneRef()
	if !active.toggleHighlightedSelection() {
		return
	}

	currentIndex := active.table.GetHighlightedRowIndex()
	lastIndex := len(active.table.GetVisibleRows()) - 1
	nextIndex := currentIndex + 1
	if nextIndex > lastIndex {
		nextIndex = lastIndex
	}

	active.table = active.table.WithHighlightedRow(nextIndex)
}

func (m *Model) openSearchModal() {
	m.searchModalVisible = true
	m.searchTargetPane = m.activePane
	m.searchInput.SetValue(m.activePaneRef().searchQuery)
	m.searchInput.SetCursor(len([]rune(m.searchInput.Value())))
	m.searchInput.Focus()
}

func (m *Model) closeSearchModal() {
	m.searchModalVisible = false
	m.searchInput.Blur()
}

func (m *Model) clearSearchHighlights() {
	m.leftPane.clearSearch(m.theme)
	m.rightPane.clearSearch(m.theme)
	m.status = ""
}

func (m *Model) applySearchInput() {
	query := m.searchInput.Value()
	target := &m.leftPane
	if m.searchTargetPane == paneRight {
		target = &m.rightPane
	}

	if query == "" {
		target.clearSearch(m.theme)
		m.status = ""
		return
	}

	expr, err := regexp.Compile(query)
	if err != nil {
		m.status = fmt.Sprintf("Search regex error: %v", err)
		return
	}

	target.setSearch(query, expr, m.theme)
	if len(target.matchIndexes) == 0 {
		m.status = "Search: no matches"
		return
	}

	target.table = target.table.WithHighlightedRow(target.matchIndexes[0])
	m.status = fmt.Sprintf("Search: %d match(es)", len(target.matchIndexes))
}

func (m *Model) moveToSearchMatch(next bool) {
	if m.activePaneRef().jumpToSearchMatch(next) {
		return
	}
	m.status = "Search: no matches"
}

func (m *Model) handleSearchModalKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.applySearchInput()
		m.closeSearchModal()
		return true, nil
	case "esc":
		m.closeSearchModal()
		return true, nil
	default:
		updated, cmd := m.searchInput.Update(msg)
		m.searchInput = updated
		return true, []tea.Cmd{cmd}
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	if m.searchModalVisible {
		return m.handleSearchModalKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return true, []tea.Cmd{tea.Quit}
	case "esc":
		m.clearSearchHighlights()
		return true, nil
	case "/":
		m.openSearchModal()
		return true, nil
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
	case " ":
		m.selectHighlightedAndAdvance()
		return true, nil
	case ".":
		m.toggleHiddenFiles()
		return true, nil
	case "n":
		m.moveToSearchMatch(true)
		return true, nil
	case "N", "shift+n":
		m.moveToSearchMatch(false)
		return true, nil
	case "G", "shift+g", "end":
		m.jumpToLastRow()
		return true, nil
	case "enter", "l":
		if err := m.activePaneRef().enterHighlightedDirectory(m.fs, m.theme); err != nil {
			m.status = err.Error()
		} else {
			m.status = ""
		}
		return true, nil
	case "backspace", "h":
		if err := m.activePaneRef().goParent(m.fs, m.theme); err != nil {
			m.status = err.Error()
		} else {
			m.status = ""
		}
		return true, nil
	default:
		return false, nil
	}
}
