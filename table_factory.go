package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

var customBorder = table.Border{
	Top:    "─",
	Left:   "│",
	Right:  "│",
	Bottom: "─",

	TopRight:    "╮",
	TopLeft:     "╭",
	BottomRight: "╯",
	BottomLeft:  "╰",

	TopJunction:    "┬",
	LeftJunction:   "├",
	RightJunction:  "┤",
	BottomJunction: "┴",
	InnerJunction:  "┼",

	InnerDivider: "│",
}

func createTable(rows []table.Row, theme appTheme) table.Model {
	flexColumns := []table.Column{
		table.NewFlexColumn(columnKeyName, "Name", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Left)),
		table.NewColumn(columnKeySize, "Size", 5).WithStyle(lipgloss.NewStyle().Align(lipgloss.Right)),
		table.NewColumn(columnKeyDate, "Date", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Right)),
		table.NewColumn(columnKeyTime, "Time", 8).WithStyle(lipgloss.NewStyle().Align(lipgloss.Right)),
	}

	keys := table.DefaultKeyMap()
	keys.RowDown.SetKeys("j", "down", "s")
	keys.RowUp.SetKeys("k", "up", "w")
	keys.RowSelectToggle.SetKeys(" ")

	return applyThemeToTable(
		table.New(flexColumns).
			WithRows(rows).
			SelectableRows(true).
			Border(customBorder).
			WithKeyMap(keys).
			WithStaticFooter("Footer!").
			WithNoPagination().
			WithSelectedText(" ", "✓").
			WithBaseStyle(lipgloss.NewStyle().Align(lipgloss.Left)).
			SortByAsc(columnKeyName),
		theme,
	)
}

func applyThemeToTable(t table.Model, theme appTheme) table.Model {
	return t.
		HeaderStyle(lipgloss.NewStyle().Foreground(theme.header).Bold(true)).
		HighlightStyle(
			lipgloss.NewStyle().
				Foreground(theme.highlightFG).
				Background(theme.highlightBG).
				Bold(true),
		).
		Border(customBorder).
		WithBaseStyle(
			lipgloss.NewStyle().
				BorderForeground(theme.border).
				Foreground(theme.text).
				Align(lipgloss.Left),
		).
		WithMissingDataIndicatorStyled(table.StyledCell{
			Style: lipgloss.NewStyle().Foreground(theme.missing),
			Data:  "",
		})
}
