package app

import "fmt"

func footerNameOrPlaceholder(name string) string {
	if name == "" {
		return "<empty>"
	}

	return name
}

func paneFooter(path string, highlightedName string) string {
	return fmt.Sprintf("%s | %s", path, footerNameOrPlaceholder(highlightedName))
}

func paneStateSuffix(p Pane) string {
	if p.isLoading {
		return " | loading..."
	}
	if p.loadErr != nil {
		return fmt.Sprintf(" | error: %v", p.loadErr)
	}
	return ""
}

func (m *Model) updateFooter() {
	leftSelected := selectedCount(m.leftPane.selected)
	rightSelected := selectedCount(m.rightPane.selected)

	m.leftPane.table = m.leftPane.table.WithStaticFooter(
		fmt.Sprintf("%s | selected: %d%s", paneFooter(m.leftPane.path, m.leftPane.highlightedName()), leftSelected, paneStateSuffix(m.leftPane)),
	)
	m.rightPane.table = m.rightPane.table.WithStaticFooter(
		fmt.Sprintf("%s | selected: %d%s", paneFooter(m.rightPane.path, m.rightPane.highlightedName()), rightSelected, paneStateSuffix(m.rightPane)),
	)
}

func selectedCount(selected map[string]bool) int {
	total := 0
	for _, isSelected := range selected {
		if isSelected {
			total++
		}
	}

	return total
}
