package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

func pressKey(t *testing.T, model Model, key tea.KeyMsg) Model {
	t.Helper()
	updated, _ := model.Update(key)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("unexpected model type: %T", updated)
	}

	return next
}

func TestFocusedPaneMovesIndependently(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	if !model.leftPane.table.GetFocused() || model.rightPane.table.GetFocused() {
		t.Fatalf("expected left pane focused initially")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if got := model.leftPane.table.GetHighlightedRowIndex(); got != 1 {
		t.Fatalf("expected left pane index 1, got %d", got)
	}
	if got := model.rightPane.table.GetHighlightedRowIndex(); got != 0 {
		t.Fatalf("expected right pane index 0, got %d", got)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyTab})
	if model.leftPane.table.GetFocused() || !model.rightPane.table.GetFocused() {
		t.Fatalf("expected right pane focused after tab")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if got := model.leftPane.table.GetHighlightedRowIndex(); got != 1 {
		t.Fatalf("expected left pane index unchanged at 1, got %d", got)
	}
	if got := model.rightPane.table.GetHighlightedRowIndex(); got != 1 {
		t.Fatalf("expected right pane index 1, got %d", got)
	}
}

func TestEnterAndParentNavigationOnFocusedPane(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a_dir")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(child, "nested.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "z_file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write root file: %v", err)
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.leftPane.path != child {
		t.Fatalf("expected left pane path %q, got %q", child, model.leftPane.path)
	}
	if model.rightPane.path != root {
		t.Fatalf("expected right pane path %q, got %q", root, model.rightPane.path)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyBackspace})
	if model.leftPane.path != root {
		t.Fatalf("expected left pane to return to %q, got %q", root, model.leftPane.path)
	}
}

func TestEnterOnFileDoesNotChangePath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a_file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.leftPane.path != root {
		t.Fatalf("expected path unchanged at %q, got %q", root, model.leftPane.path)
	}
}

func TestCapitalGMovesToLastItem(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	last := len(model.leftPane.table.GetVisibleRows()) - 1
	if got := model.leftPane.table.GetHighlightedRowIndex(); got != last {
		t.Fatalf("expected highlighted index %d, got %d", last, got)
	}
}

func TestSpaceSelectsAndMovesToNextRow(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	firstName := model.leftPane.highlightedName()
	if firstName == "" {
		t.Fatal("expected a highlighted row")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})

	if !model.leftPane.isSelected(firstName) {
		t.Fatalf("expected %q to be selected", firstName)
	}
	if got := model.leftPane.table.GetHighlightedRowIndex(); got != 1 {
		t.Fatalf("expected highlight to move to index 1, got %d", got)
	}
}

func TestSearchModalApplyHighlightsAndNavigateMatches(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha.txt", "beta.txt", "omega.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !model.searchModalVisible {
		t.Fatal("expected search modal to be visible")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.searchModalVisible {
		t.Fatal("expected search modal to be hidden after enter")
	}
	if model.leftPane.searchRegex == nil {
		t.Fatal("expected compiled regex to be set")
	}
	if len(model.leftPane.matchIndexes) != 1 {
		t.Fatalf("expected one match, got %d", len(model.leftPane.matchIndexes))
	}

	var foundStyled bool
	for _, row := range model.leftPane.table.GetVisibleRows() {
		if rowNameFromData(row.Data) != "beta.txt" {
			continue
		}

		cell, ok := row.Data[columnKeyName].(table.StyledCell)
		if !ok {
			t.Fatal("expected matched row name to be styled")
		}
		rendered, _ := cell.Data.(string)
		if !strings.Contains(rendered, "be") || !strings.Contains(rendered, "ta") || !strings.Contains(rendered, ".txt") {
			t.Fatal("expected rendered text to include matching and non-matching segments")
		}
		foundStyled = true
	}
	if !foundStyled {
		t.Fatal("expected to find styled matching row")
	}

	start := model.leftPane.table.GetHighlightedRowIndex()
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if got := model.leftPane.table.GetHighlightedRowIndex(); got != start {
		t.Fatalf("expected single-match navigation to remain at %d, got %d", start, got)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if got := model.leftPane.table.GetHighlightedRowIndex(); got != start {
		t.Fatalf("expected reverse single-match navigation to remain at %d, got %d", start, got)
	}
}

func TestEscapeClearsSearchHighlights(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha.txt", "beta.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.leftPane.searchRegex == nil {
		t.Fatal("expected search to be active before escape")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})

	if model.leftPane.searchRegex != nil {
		t.Fatal("expected search regex to be cleared")
	}
	if len(model.leftPane.matchIndexes) != 0 {
		t.Fatalf("expected no match indexes after clear, got %d", len(model.leftPane.matchIndexes))
	}

	for _, row := range model.leftPane.table.GetVisibleRows() {
		if _, ok := row.Data[columnKeyName].(table.StyledCell); ok {
			t.Fatal("expected no styled name cells after clearing search")
		}
	}
}

func TestEscapeInSearchModalCancelsModal(t *testing.T) {
	model := NewModelWithFS(OSFileSystem{}, t.TempDir())
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !model.searchModalVisible {
		t.Fatal("expected modal to open")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})

	if model.searchModalVisible {
		t.Fatal("expected modal to close on escape")
	}
	if model.leftPane.searchRegex != nil {
		t.Fatal("expected search to remain inactive after cancel")
	}
}

func TestSearchDoesNotReorderRows(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zzz.txt", "alpha.txt", "middle.txt", "beta.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := NewModelWithFS(OSFileSystem{}, root)
	before := visibleNames(model.leftPane.table.GetVisibleRows())

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	after := visibleNames(model.leftPane.table.GetVisibleRows())
	if !slices.Equal(before, after) {
		t.Fatalf("expected row order unchanged, before=%v after=%v", before, after)
	}
}

func visibleNames(rows []table.Row) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		name := rowNameFromData(row.Data)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	return names
}
