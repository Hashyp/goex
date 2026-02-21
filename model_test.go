package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
