package app

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

const (
	searchInputCharLimit = 256
	searchInputWidth     = 48
)

func newSearchInput() textinput.Model {
	input := textinput.New()
	input.Prompt = "regex> "
	input.Placeholder = "type regular expression"
	input.CharLimit = searchInputCharLimit
	input.Width = searchInputWidth
	return input
}

func rowNameFromData(data table.RowData) string {
	if raw, ok := data[columnKeyNameRaw].(string); ok {
		return raw
	}

	switch value := data[columnKeyName].(type) {
	case string:
		return value
	case table.StyledCell:
		name, _ := value.Data.(string)
		return name
	default:
		return ""
	}
}

func rowEntryIDFromData(data table.RowData) string {
	if raw, ok := data[columnKeyEntryID].(string); ok {
		return raw
	}

	return ""
}

func (p *Pane) refreshRows(theme appTheme) {
	rows := make([]table.Row, 0, len(p.entries))
	for _, entry := range p.entries {
		data := make(table.RowData, 6)
		name := entry.Name
		data[columnKeyEntryID] = entry.ID
		data[columnKeyNameRaw] = name
		if p.searchRegex != nil {
			if highlighted, matched := highlightedSearchText(name, p.searchRegex, theme); matched {
				data[columnKeyName] = table.NewStyledCell(highlighted, lipgloss.NewStyle())
			} else {
				data[columnKeyName] = name
			}
		} else {
			data[columnKeyName] = name
		}

		data[columnKeySize] = entry.TypeOrSize()

		if entry.HasModTime {
			data[columnKeyDate] = entry.ModTime.Format("2006-01-02")
			data[columnKeyTime] = entry.ModTime.Format("15:04:05")
		} else {
			data[columnKeyDate] = ""
			data[columnKeyTime] = ""
		}

		rows = append(rows, table.NewRow(data))
	}

	p.table = p.table.WithRows(rows)
	p.rebuildMatchIndexes()
}

func (p *Pane) rebuildMatchIndexes() {
	p.matchIndexes = nil
	if p.searchRegex == nil {
		return
	}

	for index, row := range p.table.GetVisibleRows() {
		if hasNonEmptyRegexMatch(p.searchRegex, rowNameFromData(row.Data)) {
			p.matchIndexes = append(p.matchIndexes, index)
		}
	}
}

func hasNonEmptyRegexMatch(expr *regexp.Regexp, value string) bool {
	if expr == nil {
		return false
	}

	indexes := expr.FindAllStringIndex(value, -1)
	for _, match := range indexes {
		if len(match) == 2 && match[1] > match[0] {
			return true
		}
	}

	return false
}

func highlightedSearchText(value string, expr *regexp.Regexp, theme appTheme) (string, bool) {
	indexes := expr.FindAllStringIndex(value, -1)
	if len(indexes) == 0 {
		return value, false
	}

	matchStyle := lipgloss.NewStyle().Foreground(theme.highlightFG).Background(theme.header).Bold(true)
	builder := strings.Builder{}
	builder.Grow(len(value) + len(indexes)*6)

	lastIndex := 0
	hasMatch := false
	for _, indexPair := range indexes {
		if len(indexPair) != 2 {
			continue
		}

		start := indexPair[0]
		end := indexPair[1]
		if end <= start {
			continue
		}

		hasMatch = true
		if start > lastIndex {
			builder.WriteString(value[lastIndex:start])
		}

		builder.WriteString(matchStyle.Render(value[start:end]))
		lastIndex = end
	}

	if !hasMatch {
		return value, false
	}

	if lastIndex < len(value) {
		builder.WriteString(value[lastIndex:])
	}

	return builder.String(), true
}
