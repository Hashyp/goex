package app

import (
	"context"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

type Pane struct {
	backend              PaneBackend
	location             Location
	path                 string
	table                table.Model
	selected             map[string]bool
	showHidden           bool
	entries              []Entry
	searchQuery          string
	searchRegex          *regexp.Regexp
	matchIndexes         []int
	isLoading            bool
	loadErr              error
	loadSeq              int
	pendingHighlightName string
}

func newPane(backend PaneBackend, theme appTheme, showHidden bool) Pane {
	selected := map[string]bool{}
	location := backend.InitialLocation()

	pane := Pane{
		backend:              backend,
		location:             location,
		path:                 backend.DisplayPath(location),
		table:                createTable([]table.Row{}, theme, selected),
		selected:             selected,
		showHidden:           showHidden,
		entries:              []Entry{},
		searchQuery:          "",
		searchRegex:          nil,
		matchIndexes:         nil,
		isLoading:            false,
		loadErr:              nil,
		loadSeq:              0,
		pendingHighlightName: "",
	}
	pane.refreshRows(theme)
	return pane
}

func (p *Pane) reset(backend PaneBackend, theme appTheme) {
	location := backend.InitialLocation()

	p.backend = backend
	p.location = location
	p.path = backend.DisplayPath(location)
	p.selected = map[string]bool{}
	p.entries = []Entry{}
	p.searchQuery = ""
	p.searchRegex = nil
	p.matchIndexes = nil
	p.isLoading = false
	p.loadErr = nil
	p.loadSeq = 0
	p.pendingHighlightName = ""
	p.table = createTable([]table.Row{}, theme, p.selected)
}

func (p *Pane) highlightedName() string {
	entry, ok := p.highlightedEntry()
	if !ok {
		return ""
	}

	return entry.Name
}

func (p *Pane) highlightedEntry() (Entry, bool) {
	highlighted := p.table.HighlightedRow()
	if highlighted.Data == nil {
		return Entry{}, false
	}

	entryID := rowEntryIDFromData(highlighted.Data)
	if entryID == "" {
		return Entry{}, false
	}

	for _, entry := range p.entries {
		if entry.ID == entryID {
			return entry, true
		}
	}

	return Entry{}, false
}

func (p *Pane) enterHighlighted(ctx context.Context) (bool, error) {
	highlighted, ok := p.highlightedEntry()
	if !ok {
		return false, nil
	}

	nextLocation, changed, err := p.backend.Enter(ctx, p.location, highlighted)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	p.location = nextLocation
	p.path = p.backend.DisplayPath(nextLocation)
	return true, nil
}

func (p *Pane) goParent() bool {
	childName := p.backend.ParentHighlightName(p.location)

	nextLocation, changed := p.backend.Parent(p.location)
	if !changed {
		return false
	}

	p.location = nextLocation
	p.path = p.backend.DisplayPath(nextLocation)
	p.pendingHighlightName = childName
	return true
}

func (p *Pane) isSelected(id string) bool {
	return p.selected[id]
}

func (p *Pane) toggleHighlightedSelection() bool {
	entry, ok := p.highlightedEntry()
	if !ok {
		return false
	}

	p.selected[entry.ID] = !p.selected[entry.ID]
	return true
}

func (p *Pane) clearSearch(theme appTheme) {
	p.searchQuery = ""
	p.searchRegex = nil
	p.refreshRows(theme)
}

func (p *Pane) setSearch(query string, expr *regexp.Regexp, theme appTheme) {
	p.searchQuery = query
	p.searchRegex = expr
	p.refreshRows(theme)
}

func (p *Pane) jumpToSearchMatch(next bool) bool {
	if len(p.matchIndexes) == 0 {
		return false
	}

	current := p.table.GetHighlightedRowIndex()
	target := p.matchIndexes[0]

	if next {
		for _, index := range p.matchIndexes {
			if index > current {
				target = index
				p.table = p.table.WithHighlightedRow(target)
				return true
			}
		}

		p.table = p.table.WithHighlightedRow(target)
		return true
	}

	target = p.matchIndexes[len(p.matchIndexes)-1]
	for i := len(p.matchIndexes) - 1; i >= 0; i-- {
		index := p.matchIndexes[i]
		if index < current {
			target = index
			break
		}
	}

	p.table = p.table.WithHighlightedRow(target)
	return true
}

func (p *Pane) highlightByName(name string) {
	if name == "" {
		return
	}

	for index, row := range p.table.GetVisibleRows() {
		if rowNameFromData(row.Data) == name {
			p.table = p.table.WithHighlightedRow(index)
			return
		}
	}
}

func (p *Pane) beginLoad(pane activePane) tea.Cmd {
	p.loadSeq++
	seq := p.loadSeq
	p.isLoading = true
	p.loadErr = nil

	backend := p.backend
	location := p.location
	showHidden := p.showHidden

	return func() tea.Msg {
		timeout := backend.LoadTimeout()
		if timeout <= 0 {
			timeout = defaultLoadTimeout
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		entries, err := backend.List(ctx, location, showHidden)
		if err != nil {
			return paneLoadErrorMsg{pane: pane, seq: seq, err: err}
		}
		return paneLoadSuccessMsg{pane: pane, seq: seq, entries: entries}
	}
}

const defaultLoadTimeout = 10 * time.Second
