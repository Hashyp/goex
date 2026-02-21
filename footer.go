package main

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

func (m *Model) updateFooter() {
	m.leftPane.table = m.leftPane.table.WithStaticFooter(
		paneFooter(m.leftPane.path, m.leftPane.highlightedName()),
	)
	m.rightPane.table = m.rightPane.table.WithStaticFooter(
		paneFooter(m.rightPane.path, m.rightPane.highlightedName()),
	)
}
