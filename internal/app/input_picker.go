package app

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) openPanePickerModal() {
	target := m.activePane
	pane := m.paneByID(target)
	choice := paneBackendChoiceFromPane(*pane)

	m.pickerModalVisible = true
	m.pickerTargetPane = target
	m.pickerChoiceIndex = findPaneBackendChoiceIndex(choice)
}

func (m *Model) closePanePickerModal() {
	m.pickerModalVisible = false
}

func (m *Model) shiftPanePickerChoice(delta int) {
	total := len(paneBackendChoices)
	if total == 0 {
		m.pickerChoiceIndex = 0
		return
	}

	next := (m.pickerChoiceIndex + delta) % total
	if next < 0 {
		next += total
	}
	m.pickerChoiceIndex = next
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
