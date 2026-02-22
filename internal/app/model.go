package app

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

type activePane int

const (
	paneLeft activePane = iota
	paneRight
)

type paneLoadSuccessMsg struct {
	pane    activePane
	seq     int
	entries []Entry
}

type paneLoadErrorMsg struct {
	pane activePane
	seq  int
	err  error
}

type initLoadMsg struct{}

type Model struct {
	leftPane           Pane
	rightPane          Pane
	activePane         activePane
	themeIndex         int
	theme              appTheme
	status             string
	width              int
	height             int
	searchModalVisible bool
	searchInput        textinput.Model
	searchTargetPane   activePane
	pickerModalVisible bool
	pickerTargetPane   activePane
	pickerChoiceIndex  int
}

func NewModel() Model {
	cwd := currentWorkingDirectory()
	leftBackend := paneBackendForChoice(paneBackendFilesystem, cwd)
	rightBackend := paneBackendForChoice(paneBackendS3, cwd)
	return NewModelWithBackends(leftBackend, rightBackend)
}

func NewModelWithFS(fs FileSystem, startPath string) Model {
	local := NewLocalBackend(fs, startPath)
	model := NewModelWithBackends(local, local)
	model.bootstrapSync()
	return model
}

func NewModelWithBackends(leftBackend PaneBackend, rightBackend PaneBackend) Model {
	themeIndex := 0
	theme := themes[themeIndex]
	showHidden := true

	leftPane := newPane(leftBackend, theme, showHidden)
	rightPane := newPane(rightBackend, theme, showHidden)

	model := Model{
		leftPane:           leftPane,
		rightPane:          rightPane,
		activePane:         paneLeft,
		themeIndex:         themeIndex,
		theme:              theme,
		status:             "",
		searchModalVisible: false,
		searchInput:        newSearchInput(),
		searchTargetPane:   paneLeft,
		pickerModalVisible: false,
		pickerTargetPane:   paneLeft,
		pickerChoiceIndex:  0,
	}

	model.setActivePane(paneLeft)
	model.updateFooter()
	return model
}

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

func (m *Model) switchPaneBackend(target activePane, choice paneBackendChoice) tea.Cmd {
	pane := m.paneByID(target)
	localStartPath := currentWorkingDirectory()
	if local, ok := pane.location.(LocalLocation); ok && local.Path != "" {
		localStartPath = local.Path
	}

	backend := paneBackendForChoice(choice, localStartPath)
	location := backend.InitialLocation()

	pane.backend = backend
	pane.location = location
	pane.path = backend.DisplayPath(location)
	pane.selected = map[string]bool{}
	pane.entries = []Entry{}
	pane.searchQuery = ""
	pane.searchRegex = nil
	pane.matchIndexes = nil
	pane.isLoading = false
	pane.loadErr = nil
	pane.loadSeq = 0
	pane.pendingHighlightName = ""
	pane.table = createTable([]table.Row{}, m.theme, pane.selected)

	m.setActivePane(m.activePane)
	m.status = fmt.Sprintf("%s pane backend: %s", paneName(target), paneBackendLabel(choice))
	return pane.beginLoad(target)
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return initLoadMsg{} }
}

func (m *Model) paneByID(id activePane) *Pane {
	if id == paneRight {
		return &m.rightPane
	}

	return &m.leftPane
}

func (m *Model) applyLoadedEntries(p *Pane, entries []Entry) {
	p.entries = entries
	p.path = p.backend.DisplayPath(p.location)
	p.loadErr = nil
	p.refreshRows(m.theme)
	if p.pendingHighlightName != "" {
		p.highlightByName(p.pendingHighlightName)
		p.pendingHighlightName = ""
	}
}

func (m *Model) bootstrapSync() {
	for _, paneID := range []activePane{paneLeft, paneRight} {
		pane := m.paneByID(paneID)
		entries, err := pane.backend.List(context.Background(), pane.location, pane.showHidden)
		if err != nil {
			pane.loadErr = err
			continue
		}
		m.applyLoadedEntries(pane, entries)
	}
	m.updateFooter()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(typed.Width, typed.Height)
	case initLoadMsg:
		cmds = append(cmds, m.leftPane.beginLoad(paneLeft), m.rightPane.beginLoad(paneRight))
	case paneLoadSuccessMsg:
		pane := m.paneByID(typed.pane)
		if typed.seq != pane.loadSeq {
			break
		}
		pane.isLoading = false
		m.applyLoadedEntries(pane, typed.entries)
		m.status = ""
	case paneLoadErrorMsg:
		pane := m.paneByID(typed.pane)
		if typed.seq != pane.loadSeq {
			break
		}
		pane.isLoading = false
		pane.loadErr = typed.err
		if len(pane.entries) == 0 {
			pane.refreshRows(m.theme)
		}
		m.status = fmt.Sprintf("%s: %v", pane.path, typed.err)
	case tea.KeyMsg:
		handled, keyCmds := m.handleKey(typed)
		cmds = append(cmds, keyCmds...)
		if !handled {
			cmds = append(cmds, m.updateActiveTable(msg)...)
		}
	default:
		cmds = append(cmds, m.updateAllTables(msg)...)
	}

	m.applyLayout()
	m.updateFooter()
	return m, tea.Batch(cmds...)
}
