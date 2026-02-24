package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func oppositePane(p activePane) activePane {
	if p == paneLeft {
		return paneRight
	}

	return paneLeft
}

func (m *Model) openCopyModal() bool {
	source := m.activePaneRef()
	entries := source.selectedCopyEntries()
	if len(entries) == 0 {
		highlighted, ok := source.highlightedEntry()
		if !ok {
			m.status = "Copy: no entry selected"
			return false
		}
		if !isCopyTargetKind(highlighted.Kind) {
			m.status = "Copy: unsupported target"
			return false
		}
		entries = append(entries, highlighted)
	}

	m.copyModal.visible = true
	m.copyModal.sourcePane = m.activePane
	m.copyModal.destinationPane = oppositePane(m.activePane)
	m.copyModal.entries = entries
	m.copyModal.progress = copyProgressState{}
	m.copyModal.plan = nil
	m.copyModal.planIndex = 0
	m.copyModal.planned = 0
	m.copyModal.result = TransferResult{}
	m.copyModal.planningErr = nil
	m.copyModal.hasResult = false
	return true
}

func (m *Model) closeCopyModal() {
	m.copyModal = copyModalState{
		visible: false,
	}
}

func copyProgressTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return copyProgressTickMsg{}
	})
}

func copyCommandContext(deadline time.Time) (context.Context, context.CancelFunc) {
	if !deadline.IsZero() {
		return context.WithDeadline(context.Background(), deadline)
	}

	return context.WithCancel(context.Background())
}

func (m *Model) startCopyProgress() []tea.Cmd {
	sourcePane := m.paneByID(m.copyModal.sourcePane)
	destinationPane := m.paneByID(m.copyModal.destinationPane)

	timeout := sourcePane.backend.LoadTimeout()
	if timeout <= 0 {
		timeout = defaultLoadTimeout
	}
	if destinationTimeout := destinationPane.backend.LoadTimeout(); destinationTimeout > timeout {
		timeout = destinationTimeout
	}

	m.copyModal.progress.inProgress = true
	m.copyModal.progress.done = 0
	m.copyModal.progress.total = 0
	m.copyModal.progress.current = ""
	m.copyModal.progress.frame = 0
	m.copyModal.progress.deadline = time.Now().Add(timeout * 4)
	m.copyModal.hasResult = false
	m.copyModal.planningErr = nil
	m.copyModal.plan = nil
	m.copyModal.planIndex = 0
	m.copyModal.result = TransferResult{}

	return []tea.Cmd{
		copyProgressTickCmd(),
		m.planCopyCmd(),
	}
}

func (m *Model) planCopyCmd() tea.Cmd {
	sourcePane := m.paneByID(m.copyModal.sourcePane)
	destinationPane := m.paneByID(m.copyModal.destinationPane)
	selected := append([]Entry(nil), m.copyModal.entries...)
	sourceID := m.copyModal.sourcePane
	destinationID := m.copyModal.destinationPane
	deadline := m.copyModal.progress.deadline

	return func() tea.Msg {
		ctx, cancel := copyCommandContext(deadline)
		defer cancel()

		enumerator, ok := sourcePane.backend.(CopyEnumerator)
		if !ok {
			return copyPlanReadyMsg{
				sourcePane:      sourceID,
				destinationPane: destinationID,
				err:             fmt.Errorf("source backend does not support copy enumeration"),
			}
		}

		plan, err := enumerator.EnumerateCopy(ctx, sourcePane.location, selected, destinationPane.location)
		if err != nil {
			return copyPlanReadyMsg{
				sourcePane:      sourceID,
				destinationPane: destinationID,
				err:             err,
			}
		}

		return copyPlanReadyMsg{
			sourcePane:      sourceID,
			destinationPane: destinationID,
			plan:            plan,
		}
	}
}

func (m *Model) nextCopyStepCmd() tea.Cmd {
	if m.copyModal.planIndex < 0 || m.copyModal.planIndex >= len(m.copyModal.plan) {
		return nil
	}
	sourcePane := m.paneByID(m.copyModal.sourcePane)
	destinationPane := m.paneByID(m.copyModal.destinationPane)
	item := m.copyModal.plan[m.copyModal.planIndex]
	deadline := m.copyModal.progress.deadline

	return func() tea.Msg {
		ctx, cancel := copyCommandContext(deadline)
		defer cancel()

		reader, ok := sourcePane.backend.(CopyReader)
		if !ok {
			return copyStepResultMsg{
				item: item,
				result: TransferResult{
					Op: TransferOpCopy,
					Failed: []TransferFailure{
						{
							PlanItem: item,
							Stage:    "open-source",
							Err:      fmt.Errorf("source backend does not support copy read"),
						},
					},
				},
			}
		}
		writer, ok := destinationPane.backend.(CopyWriter)
		if !ok {
			return copyStepResultMsg{
				item: item,
				result: TransferResult{
					Op: TransferOpCopy,
					Failed: []TransferFailure{
						{
							PlanItem: item,
							Stage:    "open-destination",
							Err:      fmt.Errorf("destination backend does not support copy write"),
						},
					},
				},
			}
		}

		result := ExecuteCopy(ctx, TransferCopyRequest{
			Plan:           []TransferPlanItem{item},
			ConflictPolicy: TransferConflictSkip,
		}, reader, writer)

		return copyStepResultMsg{
			item:   item,
			result: result,
		}
	}
}

func (m *Model) handleCopyModalKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	if m.copyModal.progress.inProgress {
		return true, nil
	}

	if m.copyModal.hasResult || m.copyModal.planningErr != nil {
		switch msg.String() {
		case "enter", "esc", "q":
			m.closeCopyModal()
			return true, nil
		default:
			return true, nil
		}
	}

	switch msg.String() {
	case "y", "Y", "shift+y":
		return true, m.startCopyProgress()
	case "n", "N", "shift+n", "esc":
		m.closeCopyModal()
		return true, nil
	default:
		return true, nil
	}
}
