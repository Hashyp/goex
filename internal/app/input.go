package app

import (
	"context"
	"fmt"
	"regexp"
	"time"

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

func (m *Model) toggleHiddenFiles() tea.Cmd {
	active := m.activePaneRef()
	active.showHidden = !active.showHidden
	if active.showHidden {
		m.status = "Hidden files: shown"
	} else {
		m.status = "Hidden files: hidden"
	}

	return active.beginLoad(m.activePane)
}

func (m *Model) reloadActivePane() tea.Cmd {
	return m.activePaneRef().beginLoad(m.activePane)
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

func (m *Model) openDeleteModal() bool {
	active := m.activePaneRef()
	entries := active.selectedDeleteEntries()
	if len(entries) == 0 {
		highlighted, ok := active.highlightedEntry()
		if !ok {
			m.status = "Delete: no entry selected"
			return false
		}
		if !isDeleteTargetKind(highlighted.Kind) {
			m.status = "Delete: only files/objects and directories are supported"
			return false
		}
		entries = append(entries, highlighted)
	}

	m.deleteModalVisible = true
	m.deleteTargetPane = m.activePane
	m.deleteTargetEntries = entries
	return true
}

func (m *Model) closeDeleteModal() {
	m.deleteModalVisible = false
	m.deleteTargetEntries = nil
	m.deleteInProgress = false
	m.deleteProgressDone = 0
	m.deleteProgressTotal = 0
	m.deleteProgressName = ""
	m.deleteProgressFrame = 0
	m.deleteProgressErrs = nil
	m.deleteProgressIDs = nil
	m.deleteDeadline = time.Time{}
}

func (m *Model) finishDeleteProgress() {
	m.deleteInProgress = false
	m.deleteProgressDone = 0
	m.deleteProgressTotal = 0
	m.deleteProgressName = ""
	m.deleteProgressFrame = 0
	m.deleteProgressErrs = nil
	m.deleteProgressIDs = nil
	m.deleteDeadline = time.Time{}
	m.deleteModalVisible = false
	m.deleteTargetEntries = nil
}

func deleteProgressTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return deleteProgressTickMsg{}
	})
}

func (m *Model) startDeleteProgress() []tea.Cmd {
	pane := m.paneByID(m.deleteTargetPane)
	timeout := pane.backend.LoadTimeout()
	if timeout <= 0 {
		timeout = defaultLoadTimeout
	}

	m.deleteInProgress = true
	m.deleteProgressDone = 0
	m.deleteProgressTotal = len(m.deleteTargetEntries)
	m.deleteProgressFrame = 0
	m.deleteProgressErrs = make([]deleteFailure, 0)
	m.deleteProgressIDs = make([]string, 0, len(m.deleteTargetEntries))
	m.deleteDeadline = time.Now().Add(timeout)
	if m.deleteProgressTotal > 0 {
		m.deleteProgressName = m.deleteTargetEntries[0].Name
	} else {
		m.deleteProgressName = ""
	}

	cmds := []tea.Cmd{deleteProgressTickCmd()}
	if m.deleteProgressTotal > 0 {
		cmds = append(cmds, m.nextDeleteStepCmd())
	} else {
		result := paneDeleteResultMsg{pane: m.deleteTargetPane}
		cmds = append(cmds, func() tea.Msg { return result })
	}

	return cmds
}

func (m *Model) nextDeleteStepCmd() tea.Cmd {
	pane := m.paneByID(m.deleteTargetPane)
	backend := pane.backend
	location := pane.location
	index := m.deleteProgressDone
	if index < 0 || index >= len(m.deleteTargetEntries) {
		return nil
	}
	entry := m.deleteTargetEntries[index]
	deadline := m.deleteDeadline

	return func() tea.Msg {
		ctx := context.Background()
		var cancel context.CancelFunc
		if !deadline.IsZero() {
			ctx, cancel = context.WithDeadline(context.Background(), deadline)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		defer cancel()

		err := backend.Delete(ctx, location, entry)
		return deleteStepMsg{entry: entry, err: err}
	}
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
	if m.pickerModalVisible {
		return m.handlePanePickerModalKey(msg)
	}

	if m.searchModalVisible {
		return m.handleSearchModalKey(msg)
	}

	if m.deleteModalVisible {
		return m.handleDeleteModalKey(msg)
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
	case "p":
		m.openPanePickerModal()
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
		return true, []tea.Cmd{m.toggleHiddenFiles()}
	case "r":
		return true, []tea.Cmd{m.reloadActivePane()}
	case "d":
		m.openDeleteModal()
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
		changed, err := m.activePaneRef().enterHighlighted(context.Background())
		if err != nil {
			m.status = err.Error()
			return true, nil
		}
		if changed {
			return true, []tea.Cmd{m.reloadActivePane()}
		}
		m.status = ""
		return true, nil
	case "backspace", "h":
		if changed := m.activePaneRef().goParent(); changed {
			m.status = ""
			return true, []tea.Cmd{m.reloadActivePane()}
		}
		m.status = ""
		return true, nil
	default:
		return false, nil
	}
}

func (m *Model) handleDeleteModalKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	if m.deleteInProgress {
		return true, nil
	}

	switch msg.String() {
	case "y", "Y", "shift+y":
		return true, m.startDeleteProgress()
	case "n", "N", "shift+n", "esc":
		m.closeDeleteModal()
		return true, nil
	default:
		return true, nil
	}
}

func (m *Model) handlePanePickerModalKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	switch msg.String() {
	case "enter":
		choice := paneBackendChoices[m.pickerChoiceIndex]
		target := m.pickerTargetPane
		m.closePanePickerModal()
		return true, []tea.Cmd{m.switchPaneBackend(target, choice)}
	case "esc":
		m.closePanePickerModal()
		return true, nil
	case "up", "k":
		m.shiftPanePickerChoice(-1)
		return true, nil
	case "down", "j", "tab":
		m.shiftPanePickerChoice(1)
		return true, nil
	case "shift+tab":
		m.shiftPanePickerChoice(-1)
		return true, nil
	default:
		return true, nil
	}
}
