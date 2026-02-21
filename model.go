package main

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type activePane int

const (
	paneLeft activePane = iota
	paneRight
)

type Model struct {
	fs                 FileSystem
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
}

func NewModel() Model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	return NewModelWithFS(OSFileSystem{}, cwd)
}

func NewModelWithFS(fs FileSystem, startPath string) Model {
	themeIndex := 0
	theme := themes[themeIndex]
	leftPane, leftErr := newPane(fs, startPath, theme)
	rightPane, rightErr := newPane(fs, startPath, theme)
	status := ""
	if leftErr != nil {
		status = leftErr.Error()
	}
	if rightErr != nil {
		status = rightErr.Error()
	}

	model := Model{
		fs:                 fs,
		leftPane:           leftPane,
		rightPane:          rightPane,
		activePane:         paneLeft,
		themeIndex:         themeIndex,
		theme:              theme,
		status:             status,
		searchModalVisible: false,
		searchInput:        newSearchInput(),
		searchTargetPane:   paneLeft,
	}

	model.setActivePane(paneLeft)
	model.updateFooter()
	return model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(typed.Width, typed.Height)
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
