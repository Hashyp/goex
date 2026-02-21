package main

import (
	"path/filepath"

	"github.com/evertras/bubble-table/table"
)

type Pane struct {
	path     string
	table    table.Model
	selected map[string]bool
}

func newPane(fs FileSystem, path string, theme appTheme) (Pane, error) {
	selected := map[string]bool{}
	rows, err := getDirAndFiles(fs, path)
	if err != nil {
		return Pane{
			path:     path,
			table:    createTable([]table.Row{}, theme, selected),
			selected: selected,
		}, err
	}

	return Pane{
		path:     path,
		table:    createTable(rows, theme, selected),
		selected: selected,
	}, nil
}

func (p *Pane) reload(fs FileSystem) error {
	rows, err := getDirAndFiles(fs, p.path)
	if err != nil {
		return err
	}

	p.table = p.table.WithRows(rows)
	return nil
}

func (p *Pane) highlightedName() string {
	highlighted := p.table.HighlightedRow()
	if highlighted.Data == nil {
		return ""
	}

	name, _ := highlighted.Data[columnKeyName].(string)
	return name
}

func (p *Pane) enterHighlightedDirectory(fs FileSystem) error {
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
	return p.reload(fs)
}

func (p *Pane) goParent(fs FileSystem) error {
	parent := filepath.Dir(p.path)
	if parent == p.path {
		return nil
	}

	p.path = parent
	return p.reload(fs)
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
