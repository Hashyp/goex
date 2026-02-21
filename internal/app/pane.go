package app

import (
	"path/filepath"
	"regexp"

	"github.com/evertras/bubble-table/table"
)

type Pane struct {
	path         string
	table        table.Model
	selected     map[string]bool
	baseRows     []table.Row
	searchQuery  string
	searchRegex  *regexp.Regexp
	matchIndexes []int
}

func newPane(fs FileSystem, path string, theme appTheme) (Pane, error) {
	selected := map[string]bool{}
	rows, err := getDirAndFiles(fs, path)
	if err != nil {
		return Pane{
			path:         path,
			table:        createTable([]table.Row{}, theme, selected),
			selected:     selected,
			baseRows:     []table.Row{},
			searchQuery:  "",
			searchRegex:  nil,
			matchIndexes: nil,
		}, err
	}

	pane := Pane{
		path:         path,
		table:        createTable([]table.Row{}, theme, selected),
		selected:     selected,
		baseRows:     rows,
		searchQuery:  "",
		searchRegex:  nil,
		matchIndexes: nil,
	}
	pane.refreshRows(theme)
	return pane, nil
}

func (p *Pane) reload(fs FileSystem, theme appTheme) error {
	rows, err := getDirAndFiles(fs, p.path)
	if err != nil {
		return err
	}

	p.baseRows = rows
	p.refreshRows(theme)
	return nil
}

func (p *Pane) highlightedName() string {
	highlighted := p.table.HighlightedRow()
	if highlighted.Data == nil {
		return ""
	}

	return rowNameFromData(highlighted.Data)
}

func (p *Pane) enterHighlightedDirectory(fs FileSystem, theme appTheme) error {
	name := p.highlightedName()
	if name == "" {
		return nil
	}

	target := filepath.Join(p.path, name)
	info, err := fs.Stat(target)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return nil
	}

	p.path = target
	return p.reload(fs, theme)
}

func (p *Pane) goParent(fs FileSystem, theme appTheme) error {
	childName := filepath.Base(p.path)
	parent := filepath.Dir(p.path)
	if parent == p.path {
		return nil
	}

	p.path = parent
	if err := p.reload(fs, theme); err != nil {
		return err
	}

	p.highlightByName(childName)
	return nil
}

func (p *Pane) isSelected(name string) bool {
	return p.selected[name]
}

func (p *Pane) toggleHighlightedSelection() bool {
	name := p.highlightedName()
	if name == "" {
		return false
	}

	p.selected[name] = !p.selected[name]
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
