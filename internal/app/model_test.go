package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

type failingRemoveFS struct {
	OSFileSystem
}

func (failingRemoveFS) Remove(string) error {
	return errors.New("delete failed")
}

func (failingRemoveFS) RemoveAll(string) error {
	return errors.New("delete failed")
}

func runCmd(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()
	current := model
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		nextCmd := queue[0]
		queue = queue[1:]
		if nextCmd == nil {
			continue
		}

		msg := nextCmd()
		if msg == nil {
			continue
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}

		updated, chained := current.Update(msg)
		nextModel, ok := updated.(Model)
		if !ok {
			t.Fatalf("unexpected model type: %T", updated)
		}
		current = nextModel
		if chained != nil {
			queue = append(queue, chained)
		}
	}

	return current
}

func initModel(t *testing.T, model Model) Model {
	t.Helper()
	if model.execProcess == nil {
		model.execProcess = func(_ *exec.Cmd, callback tea.ExecCallback) tea.Cmd {
			return func() tea.Msg { return callback(nil) }
		}
	}
	if model.editorCommand == nil {
		model.editorCommand = func(path string) (*exec.Cmd, error) {
			return exec.Command("true", path), nil
		}
	}
	return runCmd(t, model, model.Init())
}

func pressKey(t *testing.T, model Model, key tea.KeyMsg) Model {
	t.Helper()
	updated, cmd := model.Update(key)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("unexpected model type: %T", updated)
	}

	return runCmd(t, next, cmd)
}

func TestFocusedPaneMovesIndependently(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
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

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
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

func TestEnterOnFileOpensEditorAndDoesNotChangePath(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "a_file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var openedPath string
	model := NewModelWithFS(OSFileSystem{}, root)
	model.editorCommand = func(path string) (*exec.Cmd, error) {
		openedPath = path
		return exec.Command("true", path), nil
	}
	model = initModel(t, model)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.leftPane.path != root {
		t.Fatalf("expected path unchanged at %q, got %q", root, model.leftPane.path)
	}
	if openedPath != filePath {
		t.Fatalf("expected editor to open %q, got %q", filePath, openedPath)
	}
}

func TestOpenKeyOnDirectoryShowsStatusHint(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "a_dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if model.status != "Open: highlight a file/object" {
		t.Fatalf("unexpected status for directory open: %q", model.status)
	}
}

func TestEnterSkipsExecutableFileAndDoesNotLaunchEditor(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "tool")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	opened := false
	model := NewModelWithFS(OSFileSystem{}, root)
	model.editorCommand = func(path string) (*exec.Cmd, error) {
		opened = true
		return exec.Command("true", path), nil
	}
	model = initModel(t, model)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if opened {
		t.Fatal("expected editor not to be launched for enter on executable")
	}
	if model.status != "Enter: executable file skipped (use o to open)" {
		t.Fatalf("unexpected status: %q", model.status)
	}
}

func TestCapitalGMovesToLastItem(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
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

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	firstEntry, ok := model.leftPane.highlightedEntry()
	if !ok {
		t.Fatal("expected a highlighted row")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})

	if !model.leftPane.isSelected(firstEntry.ID) {
		t.Fatalf("expected %q to be selected", firstEntry.ID)
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

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
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

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
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
	model := initModel(t, NewModelWithFS(OSFileSystem{}, t.TempDir()))
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

func TestPanePickerEscapeClosesWithoutChangingPane(t *testing.T) {
	root := t.TempDir()
	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))

	beforePath := model.leftPane.path

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !model.pickerModalVisible {
		t.Fatal("expected pane picker modal to open")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.pickerModalVisible {
		t.Fatal("expected pane picker modal to close on escape")
	}
	if model.leftPane.path != beforePath {
		t.Fatalf("expected left pane path unchanged, got %q want %q", model.leftPane.path, beforePath)
	}
	if _, ok := model.leftPane.location.(LocalLocation); !ok {
		t.Fatalf("expected left pane location to remain local, got %T", model.leftPane.location)
	}
}

func TestPanePickerCanSwitchRightPaneToS3(t *testing.T) {
	root := t.TempDir()
	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !model.pickerModalVisible {
		t.Fatal("expected pane picker modal to open")
	}

	// file system -> azure -> s3
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.pickerModalVisible {
		t.Fatal("expected pane picker modal to close after apply")
	}
	if got := model.rightPane.path; got != "s3:///" {
		t.Fatalf("expected right pane s3 path, got %q", got)
	}
	loc, ok := model.rightPane.location.(S3Location)
	if !ok {
		t.Fatalf("expected right pane location to be S3, got %T", model.rightPane.location)
	}
	if loc.Mode != S3ModeBuckets {
		t.Fatalf("expected right pane in buckets mode, got %+v", loc)
	}
}

func TestPanePickerAllowsSameBackendOnBothPanes(t *testing.T) {
	root := t.TempDir()
	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))

	// Switch left pane to S3.
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Switch right pane to S3 as well.
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := model.leftPane.location.(S3Location); !ok {
		t.Fatalf("expected left pane location to be S3, got %T", model.leftPane.location)
	}
	if _, ok := model.rightPane.location.(S3Location); !ok {
		t.Fatalf("expected right pane location to be S3, got %T", model.rightPane.location)
	}
}

func TestPanePickerCanSwitchRightPaneToGCS(t *testing.T) {
	root := t.TempDir()
	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !model.pickerModalVisible {
		t.Fatal("expected pane picker modal to open")
	}

	// file system -> azure -> s3 -> gcs
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.pickerModalVisible {
		t.Fatal("expected pane picker modal to close after apply")
	}
	if got := model.rightPane.path; got != "gcs:///" {
		t.Fatalf("expected right pane gcs path, got %q", got)
	}
	loc, ok := model.rightPane.location.(GCSLocation)
	if !ok {
		t.Fatalf("expected right pane location to be GCS, got %T", model.rightPane.location)
	}
	if loc.Mode != GCSModeBuckets {
		t.Fatalf("expected right pane in buckets mode, got %+v", loc)
	}
}

func TestSearchWorksForGCSBackedPane(t *testing.T) {
	left := &fakeBackend{
		location: LocalLocation{Path: "/left"},
		entries:  []Entry{{ID: "left:one", Name: "left.txt", Kind: KindObject}},
	}
	right := &fakeBackend{
		location: GCSLocation{Mode: GCSModeBuckets},
		entries: []Entry{
			{ID: "gcs-bucket:alpha", Name: "alpha-bucket", Kind: KindGCSBucket},
			{ID: "gcs-bucket:beta", Name: "beta-bucket", Kind: KindGCSBucket},
		},
	}

	model := NewModelWithBackends(left, right)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))
	model = runCmd(t, model, model.rightPane.beginLoad(paneRight))

	model.setActivePane(paneRight)
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if model.rightPane.searchRegex == nil {
		t.Fatal("expected search regex to be active on GCS pane")
	}
	if len(model.rightPane.matchIndexes) != 1 {
		t.Fatalf("expected one search match on GCS pane, got %d", len(model.rightPane.matchIndexes))
	}
	if got := model.rightPane.highlightedName(); got != "beta-bucket" {
		t.Fatalf("expected highlighted GCS match to be beta-bucket, got %q", got)
	}
}

func TestSearchDoesNotReorderRows(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zzz.txt", "alpha.txt", "middle.txt", "beta.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
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

func TestRowsShowDirectoriesBeforeFiles(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"zeta", "alpha"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	for _, file := range []string{"beta.txt", "aardvark.txt"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	got := visibleNames(model.leftPane.table.GetVisibleRows())
	want := []string{"alpha", "zeta", "aardvark.txt", "beta.txt"}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected ordering: got=%v want=%v", got, want)
	}
}

func TestModelShowsHiddenFilesByDefault(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{".env", "visible.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	got := visibleNames(model.leftPane.table.GetVisibleRows())
	if !slices.Equal(got, []string{".env", "visible.txt"}) {
		t.Fatalf("expected hidden files visible by default, got=%v", got)
	}
}

func TestDotTogglesHiddenFilesVisibilityPerPane(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".hidden-dir"), 0o755); err != nil {
		t.Fatalf("mkdir hidden dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "visible-dir"), 0o755); err != nil {
		t.Fatalf("mkdir visible dir: %v", err)
	}
	for _, name := range []string{".env", "visible.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	leftHidden := visibleNames(model.leftPane.table.GetVisibleRows())
	if !slices.Equal(leftHidden, []string{"visible-dir", "visible.txt"}) {
		t.Fatalf("expected left pane hidden entries hidden after '.', got=%v", leftHidden)
	}
	rightUnchanged := visibleNames(model.rightPane.table.GetVisibleRows())
	if !slices.Equal(rightUnchanged, []string{".hidden-dir", "visible-dir", ".env", "visible.txt"}) {
		t.Fatalf("expected right pane unchanged after left '.', got=%v", rightUnchanged)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	rightHidden := visibleNames(model.rightPane.table.GetVisibleRows())
	if !slices.Equal(rightHidden, []string{"visible-dir", "visible.txt"}) {
		t.Fatalf("expected right pane hidden entries hidden after right '.', got=%v", rightHidden)
	}
	leftStillHidden := visibleNames(model.leftPane.table.GetVisibleRows())
	if !slices.Equal(leftStillHidden, []string{"visible-dir", "visible.txt"}) {
		t.Fatalf("expected left pane unchanged after right '.', got=%v", leftStillHidden)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	rightShown := visibleNames(model.rightPane.table.GetVisibleRows())
	if !slices.Equal(rightShown, []string{".hidden-dir", "visible-dir", ".env", "visible.txt"}) {
		t.Fatalf("expected right pane hidden entries shown after second right '.', got=%v", rightShown)
	}
}

func TestBackToParentHighlightsVisitedDirectory(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "alpha")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "beta.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "inside.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	if got := model.leftPane.highlightedName(); got != "alpha" {
		t.Fatalf("expected initial highlight to be alpha, got %q", got)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if model.leftPane.path != targetDir {
		t.Fatalf("expected to enter %q, got %q", targetDir, model.leftPane.path)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if model.leftPane.path != root {
		t.Fatalf("expected to return to %q, got %q", root, model.leftPane.path)
	}
	if got := model.leftPane.highlightedName(); got != "alpha" {
		t.Fatalf("expected highlight restored to alpha, got %q", got)
	}
}

func TestDeleteModalCancelsOnEsc(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.deleteModal.visible {
		t.Fatal("expected delete modal to open")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.deleteModal.visible {
		t.Fatal("expected delete modal to close on escape")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file to remain after cancel: %v", err)
	}
}

func TestDeleteConfirmRemovesFileAndReloadsPane(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if model.deleteModal.visible {
		t.Fatal("expected delete modal to close after confirmation")
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, stat err=%v", err)
	}
	if got := len(model.leftPane.table.GetVisibleRows()); got != 0 {
		t.Fatalf("expected pane to reload without deleted row, got %d rows", got)
	}
}

func TestDeleteDirectoryOpensModalAndDeletesRecursively(t *testing.T) {
	root := t.TempDir()
	alpha := filepath.Join(root, "alpha")
	if err := os.Mkdir(alpha, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alpha, "nested.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.deleteModal.visible {
		t.Fatal("expected delete modal to open for directory")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if model.deleteModal.visible {
		t.Fatal("expected delete modal to close after confirmation")
	}
	if _, err := os.Stat(alpha); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected directory to be deleted recursively, stat err=%v", err)
	}
}

func TestDeleteErrorIsShownInStatus(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	model := initModel(t, NewModelWithFS(failingRemoveFS{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if model.deleteModal.visible {
		t.Fatal("expected delete modal to close on error")
	}
	if model.status != "Deleted 0 item(s), failed 1: \"alpha.txt\": delete failed" {
		t.Fatalf("unexpected status: %q", model.status)
	}
	if got := len(model.leftPane.table.GetVisibleRows()); got != 1 {
		t.Fatalf("expected rows to remain after delete error, got %d", got)
	}
}

func TestDeleteConfirmRemovesSelectedFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	model := initModel(t, NewModelWithFS(OSFileSystem{}, root))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.deleteModal.visible {
		t.Fatal("expected delete modal to open for selected files")
	}
	if got := len(model.deleteModal.entries); got != 2 {
		t.Fatalf("expected two delete targets, got %d", got)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if model.deleteModal.visible {
		t.Fatal("expected delete modal to close after confirmation")
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %q to be deleted, stat err=%v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "c.txt")); err != nil {
		t.Fatalf("expected c.txt to remain: %v", err)
	}

	names := visibleNames(model.leftPane.table.GetVisibleRows())
	if !slices.Equal(names, []string{"c.txt"}) {
		t.Fatalf("unexpected rows after deleting selection: %v", names)
	}
	if got := selectedCount(model.leftPane.selected); got != 0 {
		t.Fatalf("expected selected count to clear after delete, got %d", got)
	}
}

func TestCopyModalCancelsOnEsc(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "alpha.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	dst := t.TempDir()

	model := initModel(t, NewModelWithBackends(
		NewLocalBackend(OSFileSystem{}, src),
		NewLocalBackend(OSFileSystem{}, dst),
	))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !model.copyModal.visible {
		t.Fatal("expected copy modal to open")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.copyModal.visible {
		t.Fatal("expected copy modal to close on escape")
	}
}

func TestCopyConfirmCopiesHighlightedFileToOppositePane(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "alpha.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	dst := t.TempDir()

	model := initModel(t, NewModelWithBackends(
		NewLocalBackend(OSFileSystem{}, src),
		NewLocalBackend(OSFileSystem{}, dst),
	))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if !model.copyModal.visible {
		t.Fatal("expected copy result modal to remain visible")
	}
	if !model.copyModal.hasResult {
		t.Fatal("expected copy result state after confirmation")
	}
	if len(model.copyModal.result.Copied) != 1 {
		t.Fatalf("expected one copied item, got %d", len(model.copyModal.result.Copied))
	}

	copiedPath := filepath.Join(dst, "alpha.txt")
	got, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if string(got) != "alpha" {
		t.Fatalf("unexpected copied file contents: %q", string(got))
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.copyModal.visible {
		t.Fatal("expected copy modal to close after result acknowledgment")
	}
}

func TestMoveModalCancelsOnEsc(t *testing.T) {
	src := t.TempDir()
	sourcePath := filepath.Join(src, "alpha.txt")
	if err := os.WriteFile(sourcePath, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	dst := t.TempDir()

	model := initModel(t, NewModelWithBackends(
		NewLocalBackend(OSFileSystem{}, src),
		NewLocalBackend(OSFileSystem{}, dst),
	))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if !model.moveModal.visible {
		t.Fatal("expected move modal to open")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.moveModal.visible {
		t.Fatal("expected move modal to close on escape")
	}
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("expected source file to remain after cancel: %v", err)
	}
}

func TestMoveConfirmMovesHighlightedFileToOppositePane(t *testing.T) {
	src := t.TempDir()
	sourcePath := filepath.Join(src, "alpha.txt")
	if err := os.WriteFile(sourcePath, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	dst := t.TempDir()

	model := initModel(t, NewModelWithBackends(
		NewLocalBackend(OSFileSystem{}, src),
		NewLocalBackend(OSFileSystem{}, dst),
	))
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if !model.moveModal.visible {
		t.Fatal("expected move result modal to remain visible")
	}
	if !model.moveModal.hasResult {
		t.Fatal("expected move result state after confirmation")
	}
	if len(model.moveModal.result.Copied) != 1 {
		t.Fatalf("expected one moved item, got %d", len(model.moveModal.result.Copied))
	}

	movedPath := filepath.Join(dst, "alpha.txt")
	got, err := os.ReadFile(movedPath)
	if err != nil {
		t.Fatalf("read moved file: %v", err)
	}
	if string(got) != "alpha" {
		t.Fatalf("unexpected moved file contents: %q", string(got))
	}
	if _, err := os.Stat(sourcePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected source file removed after move, stat err=%v", err)
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.moveModal.visible {
		t.Fatal("expected move modal to close after result acknowledgment")
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
