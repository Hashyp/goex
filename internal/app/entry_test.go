package app

import (
	"slices"
	"testing"
	"time"
)

func TestSortEntriesDirectoriesFirstThenAlphabetical(t *testing.T) {
	entries := []Entry{
		{Name: "z.txt", Kind: KindBlob},
		{Name: "beta", Kind: KindDirectory},
		{Name: "alpha", Kind: KindDirectory},
		{Name: "a.txt", Kind: KindBlob},
		{Name: "cont", Kind: KindContainer},
	}

	sortEntries(entries)

	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name)
	}

	want := []string{"alpha", "beta", "cont", "a.txt", "z.txt"}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected order: got=%v want=%v", got, want)
	}
}

func TestRefreshRowsMapsEntryFieldsToColumns(t *testing.T) {
	theme := themes[0]
	pane := newPane(NewStaticErrorBackend(nil), theme, true)
	modTime := time.Date(2026, 2, 21, 8, 15, 0, 0, time.UTC)
	pane.entries = []Entry{
		{
			ID:         "dir:docs",
			Name:       "docs",
			FullPath:   "docs",
			Kind:       KindDirectory,
			HasModTime: false,
		},
		{
			ID:         "blob:file.txt",
			Name:       "file.txt",
			FullPath:   "file.txt",
			Kind:       KindBlob,
			SizeBytes:  1536,
			ModTime:    modTime,
			HasModTime: true,
		},
	}

	pane.refreshRows(theme)
	rows := pane.table.GetVisibleRows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	first := rows[0].Data
	if rowEntryIDFromData(first) != "dir:docs" {
		t.Fatalf("unexpected first row id: %q", rowEntryIDFromData(first))
	}
	if got, _ := first[columnKeySize].(string); got != "<DIR>" {
		t.Fatalf("expected directory size marker, got %q", got)
	}
	if got, _ := first[columnKeyDate].(string); got != "" {
		t.Fatalf("expected empty date for missing mod time, got %q", got)
	}

	second := rows[1].Data
	if rowEntryIDFromData(second) != "blob:file.txt" {
		t.Fatalf("unexpected second row id: %q", rowEntryIDFromData(second))
	}
	if got, _ := second[columnKeySize].(string); got != "1.5K" {
		t.Fatalf("expected formatted size 1.5K, got %q", got)
	}
	if got, _ := second[columnKeyDate].(string); got != "2026-02-21" {
		t.Fatalf("unexpected date value: %q", got)
	}
}

func TestToggleHighlightedSelectionUsesEntryID(t *testing.T) {
	theme := themes[0]
	pane := newPane(NewStaticErrorBackend(nil), theme, true)
	pane.entries = []Entry{
		{ID: "container:foo", Name: "foo", Kind: KindContainer},
		{ID: "dir:foo", Name: "foo", Kind: KindDirectory},
	}
	pane.refreshRows(theme)

	if !pane.toggleHighlightedSelection() {
		t.Fatal("expected highlighted row selection toggle to succeed")
	}
	if !pane.isSelected("container:foo") {
		t.Fatal("expected selection key to use entry id for first row")
	}
	if pane.isSelected("dir:foo") {
		t.Fatal("did not expect second row with same name to be selected")
	}
}
