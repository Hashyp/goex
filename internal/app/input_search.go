package app

import (
	"fmt"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
)

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
