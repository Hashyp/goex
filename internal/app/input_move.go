package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) openMoveModal() bool {
	source := m.activePaneRef()
	entries := source.selectedCopyEntries()
	if len(entries) == 0 {
		highlighted, ok := source.highlightedEntry()
		if !ok {
			m.status = "Move: no entry selected"
			return false
		}
		if !isCopyTargetKind(highlighted.Kind) {
			m.status = "Move: unsupported target"
			return false
		}
		entries = append(entries, highlighted)
	}

	m.moveModal.visible = true
	m.moveModal.sourcePane = m.activePane
	m.moveModal.destinationPane = oppositePane(m.activePane)
	m.moveModal.entries = entries
	m.moveModal.progress = moveProgressState{}
	m.moveModal.plan = nil
	m.moveModal.planIndex = 0
	m.moveModal.planned = 0
	m.moveModal.result = TransferResult{}
	m.moveModal.planningErr = nil
	m.moveModal.hasResult = false
	return true
}

func (m *Model) closeMoveModal() {
	m.moveModal = moveModalState{
		visible: false,
	}
}

func moveProgressTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return moveProgressTickMsg{}
	})
}

func (m *Model) startMoveProgress() []tea.Cmd {
	sourcePane := m.paneByID(m.moveModal.sourcePane)
	destinationPane := m.paneByID(m.moveModal.destinationPane)

	timeout := sourcePane.backend.LoadTimeout()
	if timeout <= 0 {
		timeout = defaultLoadTimeout
	}
	if destinationTimeout := destinationPane.backend.LoadTimeout(); destinationTimeout > timeout {
		timeout = destinationTimeout
	}

	m.moveModal.progress.inProgress = true
	m.moveModal.progress.done = 0
	m.moveModal.progress.total = 0
	m.moveModal.progress.current = ""
	m.moveModal.progress.frame = 0
	m.moveModal.progress.deadline = time.Now().Add(timeout * 4)
	m.moveModal.hasResult = false
	m.moveModal.planningErr = nil
	m.moveModal.plan = nil
	m.moveModal.planIndex = 0
	m.moveModal.result = TransferResult{}

	return []tea.Cmd{
		moveProgressTickCmd(),
		m.planMoveCmd(),
	}
}

func (m *Model) planMoveCmd() tea.Cmd {
	sourcePane := m.paneByID(m.moveModal.sourcePane)
	destinationPane := m.paneByID(m.moveModal.destinationPane)
	selected := append([]Entry(nil), m.moveModal.entries...)
	sourceID := m.moveModal.sourcePane
	destinationID := m.moveModal.destinationPane
	deadline := m.moveModal.progress.deadline

	return func() tea.Msg {
		ctx, cancel := copyCommandContext(deadline)
		defer cancel()

		enumerator, ok := sourcePane.backend.(CopyEnumerator)
		if !ok {
			return movePlanReadyMsg{
				sourcePane:      sourceID,
				destinationPane: destinationID,
				err:             fmt.Errorf("source backend does not support move enumeration"),
			}
		}

		plan, err := enumerator.EnumerateCopy(ctx, sourcePane.location, selected, destinationPane.location)
		if err != nil {
			return movePlanReadyMsg{
				sourcePane:      sourceID,
				destinationPane: destinationID,
				err:             err,
			}
		}

		return movePlanReadyMsg{
			sourcePane:      sourceID,
			destinationPane: destinationID,
			plan:            plan,
		}
	}
}

func (m *Model) nextMoveStepCmd() tea.Cmd {
	if m.moveModal.planIndex < 0 || m.moveModal.planIndex >= len(m.moveModal.plan) {
		return nil
	}
	sourcePane := m.paneByID(m.moveModal.sourcePane)
	destinationPane := m.paneByID(m.moveModal.destinationPane)
	item := m.moveModal.plan[m.moveModal.planIndex]
	deadline := m.moveModal.progress.deadline

	return func() tea.Msg {
		ctx, cancel := copyCommandContext(deadline)
		defer cancel()

		reader, ok := sourcePane.backend.(CopyReader)
		if !ok {
			return moveStepResultMsg{
				item: item,
				result: TransferResult{
					Op: TransferOpMove,
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
		deleter, ok := sourcePane.backend.(CopySourceDeleter)
		if !ok {
			return moveStepResultMsg{
				item: item,
				result: TransferResult{
					Op: TransferOpMove,
					Failed: []TransferFailure{
						{
							PlanItem: item,
							Stage:    "delete-source",
							Err:      fmt.Errorf("source backend does not support source delete for move"),
						},
					},
				},
			}
		}
		writer, ok := destinationPane.backend.(CopyWriter)
		if !ok {
			return moveStepResultMsg{
				item: item,
				result: TransferResult{
					Op: TransferOpMove,
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

		result := ExecuteMove(ctx, TransferMoveRequest{
			Plan:           []TransferPlanItem{item},
			ConflictPolicy: TransferConflictSkip,
		}, struct {
			CopyReader
			CopySourceDeleter
		}{
			CopyReader:        reader,
			CopySourceDeleter: deleter,
		}, writer)

		return moveStepResultMsg{
			item:   item,
			result: result,
		}
	}
}

func (m *Model) handleMoveModalKey(msg tea.KeyMsg) (handled bool, cmds []tea.Cmd) {
	if m.moveModal.progress.inProgress {
		return true, nil
	}

	if m.moveModal.hasResult || m.moveModal.planningErr != nil {
		switch msg.String() {
		case "enter", "esc", "q":
			m.closeMoveModal()
			return true, nil
		default:
			return true, nil
		}
	}

	switch msg.String() {
	case "y", "Y", "shift+y":
		return true, m.startMoveProgress()
	case "n", "N", "shift+n", "esc":
		m.closeMoveModal()
		return true, nil
	default:
		return true, nil
	}
}
