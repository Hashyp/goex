package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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

	m.deleteModal.visible = true
	m.deleteModal.targetPane = m.activePane
	m.deleteModal.entries = entries
	return true
}

func (m *Model) closeDeleteModal() {
	m.deleteModal.visible = false
	m.deleteModal.entries = nil
	m.deleteModal.progress = deleteProgressState{}
}

func (m *Model) finishDeleteProgress() {
	m.deleteModal.visible = false
	m.deleteModal.entries = nil
	m.deleteModal.progress = deleteProgressState{}
}

func deleteProgressTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return deleteProgressTickMsg{}
	})
}

func (m *Model) startDeleteProgress() []tea.Cmd {
	pane := m.paneByID(m.deleteModal.targetPane)
	timeout := pane.backend.LoadTimeout()
	if timeout <= 0 {
		timeout = defaultLoadTimeout
	}

	m.deleteModal.progress.inProgress = true
	m.deleteModal.progress.done = 0
	m.deleteModal.progress.total = len(m.deleteModal.entries)
	m.deleteModal.progress.frame = 0
	m.deleteModal.progress.errs = make([]deleteFailure, 0)
	m.deleteModal.progress.ids = make([]string, 0, len(m.deleteModal.entries))
	m.deleteModal.progress.deadline = time.Now().Add(timeout)
	if m.deleteModal.progress.total > 0 {
		m.deleteModal.progress.current = m.deleteModal.entries[0].Name
	} else {
		m.deleteModal.progress.current = ""
	}

	cmds := []tea.Cmd{deleteProgressTickCmd()}
	if m.deleteModal.progress.total > 0 {
		cmds = append(cmds, m.nextDeleteStepCmd())
	} else {
		result := paneDeleteResultMsg{pane: m.deleteModal.targetPane}
		cmds = append(cmds, func() tea.Msg { return result })
	}

	return cmds
}

func (m *Model) nextDeleteStepCmd() tea.Cmd {
	pane := m.paneByID(m.deleteModal.targetPane)
	backend := pane.backend
	location := pane.location
	index := m.deleteModal.progress.done
	if index < 0 || index >= len(m.deleteModal.entries) {
		return nil
	}
	entry := m.deleteModal.entries[index]
	deadline := m.deleteModal.progress.deadline

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

func (m *Model) handleDeleteModalKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	if m.deleteModal.progress.inProgress {
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
