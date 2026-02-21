package app

import "github.com/evertras/bubble-table/table"

func (m *Model) resize(width, height int) {
	m.width = width
	m.height = height
	m.applyLayout()
}

func (m *Model) paneHeight() int {
	return max(1, m.height)
}

func (m *Model) applyLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	paneWidth := max(30, m.width/2)
	paneHeight := m.paneHeight()

	m.leftPane.table = table.Model.WithTargetWidth(m.leftPane.table, paneWidth)
	m.rightPane.table = table.Model.WithTargetWidth(m.rightPane.table, paneWidth)
	m.leftPane.table = m.leftPane.table.WithMinimumHeight(paneHeight)
	m.rightPane.table = m.rightPane.table.WithMinimumHeight(paneHeight)
}
