package app

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

type paneDeleteResultMsg struct {
	pane       activePane
	deletedIDs []string
	failed     []deleteFailure
}

type deleteStepMsg struct {
	entry Entry
	err   error
}

type deleteProgressTickMsg struct{}
type copyProgressTickMsg struct{}
type moveProgressTickMsg struct{}
type editorOpenFailedMsg struct {
	err error
}

type editorPreparedMsg struct {
	session externalEditSession
	cmd     *exec.Cmd
}

type editorProcessDoneMsg struct {
	session externalEditSession
	err     error
}

type editorSyncDoneMsg struct {
	pane     activePane
	entry    string
	uploaded bool
	err      error
}

type deleteFailure struct {
	name string
	err  error
}

type deleteProgressState struct {
	inProgress bool
	done       int
	total      int
	current    string
	frame      int
	errs       []deleteFailure
	ids        []string
	deadline   time.Time
}

type deleteModalState struct {
	visible    bool
	targetPane activePane
	entries    []Entry
	progress   deleteProgressState
}

type copyPlanReadyMsg struct {
	sourcePane      activePane
	destinationPane activePane
	plan            []TransferPlanItem
	err             error
}

type copyStepResultMsg struct {
	item   TransferPlanItem
	result TransferResult
}

type movePlanReadyMsg struct {
	sourcePane      activePane
	destinationPane activePane
	plan            []TransferPlanItem
	err             error
}

type moveStepResultMsg struct {
	item   TransferPlanItem
	result TransferResult
}

type copyProgressState struct {
	inProgress bool
	done       int
	total      int
	current    string
	frame      int
	deadline   time.Time
}

type copyModalState struct {
	visible         bool
	sourcePane      activePane
	destinationPane activePane
	entries         []Entry
	plan            []TransferPlanItem
	planIndex       int
	progress        copyProgressState
	planned         int
	result          TransferResult
	planningErr     error
	hasResult       bool
}

type moveProgressState struct {
	inProgress bool
	done       int
	total      int
	current    string
	frame      int
	deadline   time.Time
}

type moveModalState struct {
	visible         bool
	sourcePane      activePane
	destinationPane activePane
	entries         []Entry
	plan            []TransferPlanItem
	planIndex       int
	progress        moveProgressState
	planned         int
	result          TransferResult
	planningErr     error
	hasResult       bool
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
	deleteModal        deleteModalState
	copyModal          copyModalState
	moveModal          moveModalState
	pickerModalVisible bool
	pickerTargetPane   activePane
	pickerChoiceIndex  int
	execProcess        func(*exec.Cmd, tea.ExecCallback) tea.Cmd
	editorCommand      func(path string) (*exec.Cmd, error)
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
		deleteModal: deleteModalState{
			visible:    false,
			targetPane: paneLeft,
			entries:    nil,
			progress: deleteProgressState{
				inProgress: false,
				done:       0,
				total:      0,
				current:    "",
				frame:      0,
				errs:       nil,
				ids:        nil,
				deadline:   time.Time{},
			},
		},
		copyModal: copyModalState{
			visible:         false,
			sourcePane:      paneLeft,
			destinationPane: paneRight,
			entries:         nil,
			progress: copyProgressState{
				inProgress: false,
				done:       0,
				total:      0,
				current:    "",
				frame:      0,
				deadline:   time.Time{},
			},
			plan:        nil,
			planIndex:   0,
			planned:     0,
			result:      TransferResult{},
			planningErr: nil,
			hasResult:   false,
		},
		moveModal: moveModalState{
			visible:         false,
			sourcePane:      paneLeft,
			destinationPane: paneRight,
			entries:         nil,
			progress: moveProgressState{
				inProgress: false,
				done:       0,
				total:      0,
				current:    "",
				frame:      0,
				deadline:   time.Time{},
			},
			plan:        nil,
			planIndex:   0,
			planned:     0,
			result:      TransferResult{},
			planningErr: nil,
			hasResult:   false,
		},
		pickerModalVisible: false,
		pickerTargetPane:   paneLeft,
		pickerChoiceIndex:  0,
		execProcess:        tea.ExecProcess,
		editorCommand:      buildEditorCommand,
	}

	model.setActivePane(paneLeft)
	model.updateFooter()
	return model
}

func (m *Model) switchPaneBackend(target activePane, choice paneBackendChoice) tea.Cmd {
	pane := m.paneByID(target)
	localStartPath := currentWorkingDirectory()
	if local, ok := pane.location.(LocalLocation); ok && local.Path != "" {
		localStartPath = local.Path
	}

	backend := paneBackendForChoice(choice, localStartPath)
	pane.reset(backend, m.theme)

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
	case paneDeleteResultMsg:
		pane := m.paneByID(typed.pane)
		pane.clearSelected(typed.deletedIDs)
		if len(typed.failed) == 0 {
			m.status = ""
		} else {
			first := typed.failed[0]
			m.status = fmt.Sprintf("Deleted %d item(s), failed %d: %q: %v", len(typed.deletedIDs), len(typed.failed), first.name, first.err)
		}
		if len(typed.deletedIDs) == 0 {
			break
		}
		cmds = append(cmds, pane.beginLoad(typed.pane))
	case copyPlanReadyMsg:
		if !m.copyModal.visible || !m.copyModal.progress.inProgress {
			break
		}
		if typed.err != nil {
			m.copyModal.progress.inProgress = false
			m.copyModal.planningErr = typed.err
			m.copyModal.hasResult = true
			m.status = fmt.Sprintf("Copy failed: %v", typed.err)
			break
		}

		m.copyModal.plan = typed.plan
		m.copyModal.planIndex = 0
		m.copyModal.progress.total = len(typed.plan)
		m.copyModal.planned = len(typed.plan)
		m.copyModal.result = TransferResult{
			Op:      TransferOpCopy,
			Copied:  make([]TransferResultItem, 0, len(typed.plan)),
			Skipped: make([]TransferPlanItem, 0),
			Failed:  make([]TransferFailure, 0),
		}
		if len(typed.plan) == 0 {
			m.copyModal.progress.inProgress = false
			m.copyModal.hasResult = true
			m.status = "Copy complete: copied 0, skipped 0, failed 0"
			break
		}

		m.copyModal.progress.current = typed.plan[0].Source.Display
		cmds = append(cmds, m.nextCopyStepCmd())
	case copyStepResultMsg:
		if !m.copyModal.visible || !m.copyModal.progress.inProgress {
			break
		}

		m.copyModal.result = mergeTransferResults(m.copyModal.result, typed.result)
		m.copyModal.progress.done++
		m.copyModal.planIndex++

		if m.copyModal.planIndex < len(m.copyModal.plan) {
			m.copyModal.progress.current = m.copyModal.plan[m.copyModal.planIndex].Source.Display
			cmds = append(cmds, m.nextCopyStepCmd())
			break
		}

		m.copyModal.progress.inProgress = false
		m.copyModal.hasResult = true
		m.status = fmt.Sprintf(
			"Copy complete: copied %d, skipped %d, failed %d",
			len(m.copyModal.result.Copied),
			len(m.copyModal.result.Skipped),
			len(m.copyModal.result.Failed),
		)
		if len(m.copyModal.result.Copied) > 0 {
			cmds = append(cmds, m.paneByID(m.copyModal.destinationPane).beginLoad(m.copyModal.destinationPane))
		}
	case movePlanReadyMsg:
		if !m.moveModal.visible || !m.moveModal.progress.inProgress {
			break
		}
		if typed.err != nil {
			m.moveModal.progress.inProgress = false
			m.moveModal.planningErr = typed.err
			m.moveModal.hasResult = true
			m.status = fmt.Sprintf("Move failed: %v", typed.err)
			break
		}

		m.moveModal.plan = typed.plan
		m.moveModal.planIndex = 0
		m.moveModal.progress.total = len(typed.plan)
		m.moveModal.planned = len(typed.plan)
		m.moveModal.result = TransferResult{
			Op:      TransferOpMove,
			Copied:  make([]TransferResultItem, 0, len(typed.plan)),
			Skipped: make([]TransferPlanItem, 0),
			Failed:  make([]TransferFailure, 0),
		}
		if len(typed.plan) == 0 {
			m.moveModal.progress.inProgress = false
			m.moveModal.hasResult = true
			m.status = "Move complete: moved 0, skipped 0, failed 0"
			break
		}

		m.moveModal.progress.current = typed.plan[0].Source.Display
		cmds = append(cmds, m.nextMoveStepCmd())
	case moveStepResultMsg:
		if !m.moveModal.visible || !m.moveModal.progress.inProgress {
			break
		}

		m.moveModal.result = mergeTransferResults(m.moveModal.result, typed.result)
		m.moveModal.progress.done++
		m.moveModal.planIndex++

		if m.moveModal.planIndex < len(m.moveModal.plan) {
			m.moveModal.progress.current = m.moveModal.plan[m.moveModal.planIndex].Source.Display
			cmds = append(cmds, m.nextMoveStepCmd())
			break
		}

		m.moveModal.progress.inProgress = false
		m.moveModal.hasResult = true
		m.status = fmt.Sprintf(
			"Move complete: moved %d, skipped %d, failed %d",
			len(m.moveModal.result.Copied),
			len(m.moveModal.result.Skipped),
			len(m.moveModal.result.Failed),
		)
		if len(m.moveModal.result.Copied) > 0 {
			cmds = append(cmds, m.paneByID(m.moveModal.sourcePane).beginLoad(m.moveModal.sourcePane))
			cmds = append(cmds, m.paneByID(m.moveModal.destinationPane).beginLoad(m.moveModal.destinationPane))
		}
	case deleteStepMsg:
		if m.deleteModal.progress.inProgress {
			if typed.err != nil {
				m.deleteModal.progress.errs = append(m.deleteModal.progress.errs, deleteFailure{name: typed.entry.Name, err: typed.err})
			} else {
				m.deleteModal.progress.ids = append(m.deleteModal.progress.ids, typed.entry.ID)
			}
			m.deleteModal.progress.done++
			if m.deleteModal.progress.done < m.deleteModal.progress.total {
				m.deleteModal.progress.current = m.deleteModal.entries[m.deleteModal.progress.done].Name
				cmds = append(cmds, m.nextDeleteStepCmd())
				break
			}

			result := paneDeleteResultMsg{
				pane:       m.deleteModal.targetPane,
				deletedIDs: m.deleteModal.progress.ids,
				failed:     m.deleteModal.progress.errs,
			}
			m.finishDeleteProgress()
			cmds = append(cmds, func() tea.Msg { return result })
		}
	case deleteProgressTickMsg:
		if m.deleteModal.progress.inProgress {
			m.deleteModal.progress.frame++
			cmds = append(cmds, deleteProgressTickCmd())
		}
	case copyProgressTickMsg:
		if m.copyModal.progress.inProgress {
			m.copyModal.progress.frame++
			cmds = append(cmds, copyProgressTickCmd())
		}
	case moveProgressTickMsg:
		if m.moveModal.progress.inProgress {
			m.moveModal.progress.frame++
			cmds = append(cmds, moveProgressTickCmd())
		}
	case editorOpenFailedMsg:
		m.status = fmt.Sprintf("Open failed: %v", typed.err)
	case editorPreparedMsg:
		if typed.cmd == nil {
			break
		}
		m.status = fmt.Sprintf("Opening %q...", typed.session.entryName)
		cmds = append(cmds, m.execEditorProcessCmd(typed.cmd, func(err error) tea.Msg {
			return editorProcessDoneMsg{
				session: typed.session,
				err:     err,
			}
		}))
	case editorProcessDoneMsg:
		if typed.err != nil {
			cmds = append(cmds, m.cleanupTempOpenFileCmd(typed.session))
			m.status = fmt.Sprintf("Open failed for %q: %v", typed.session.entryName, typed.err)
			break
		}
		cmds = append(cmds, m.syncOpenFileChangesCmd(typed.session))
	case editorSyncDoneMsg:
		if typed.err != nil {
			m.status = fmt.Sprintf("Open sync failed for %q: %v", typed.entry, typed.err)
			break
		}
		if typed.uploaded {
			m.status = fmt.Sprintf("Saved changes to %q", typed.entry)
			cmds = append(cmds, m.paneByID(typed.pane).beginLoad(typed.pane))
			break
		}
		m.status = ""
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
