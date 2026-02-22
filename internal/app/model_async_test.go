package app

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeBackend struct {
	location  Location
	entries   []Entry
	failCount int
	calls     int
}

func (b *fakeBackend) InitialLocation() Location {
	return b.location
}

func (b *fakeBackend) List(_ context.Context, _ Location, _ bool) ([]Entry, error) {
	b.calls++
	if b.failCount > 0 {
		b.failCount--
		return nil, errors.New("list failed")
	}
	return b.entries, nil
}

func (b *fakeBackend) Enter(_ context.Context, state Location, _ Entry) (Location, bool, error) {
	return state, false, nil
}

func (b *fakeBackend) Parent(state Location) (Location, bool) {
	return state, false
}

func (b *fakeBackend) DisplayPath(_ Location) string {
	return "fake:/"
}

func TestModelInitKeepsRunningWhenRightPaneLoadFails(t *testing.T) {
	left := &fakeBackend{location: LocalLocation{Path: "/left"}, entries: []Entry{{ID: "left:file", Name: "file", Kind: KindBlob}}}
	right := &fakeBackend{location: AzureLocation{Mode: AzureModeContainers}, failCount: 1}

	model := NewModelWithBackends(left, right)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))
	model = runCmd(t, model, model.rightPane.beginLoad(paneRight))

	if len(model.leftPane.table.GetVisibleRows()) != 1 {
		t.Fatalf("expected left pane rows loaded, got %d", len(model.leftPane.table.GetVisibleRows()))
	}
	if len(model.rightPane.table.GetVisibleRows()) != 0 {
		t.Fatalf("expected right pane empty on initial failure, got %d", len(model.rightPane.table.GetVisibleRows()))
	}
	if model.rightPane.loadErr == nil {
		t.Fatal("expected right pane error state to be set")
	}
}

func TestRetryReloadsFailedPane(t *testing.T) {
	left := &fakeBackend{location: LocalLocation{Path: "/left"}, entries: []Entry{{ID: "left:file", Name: "file", Kind: KindBlob}}}
	right := &fakeBackend{
		location:  AzureLocation{Mode: AzureModeContainers},
		failCount: 1,
		entries:   []Entry{{ID: "container:goex-dev", Name: "goex-dev", Kind: KindContainer}},
	}

	model := NewModelWithBackends(left, right)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))
	model = runCmd(t, model, model.rightPane.beginLoad(paneRight))
	if model.rightPane.loadErr == nil {
		t.Fatal("expected initial load error")
	}

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = pressKey(t, model, teaKey('r'))

	if model.rightPane.loadErr != nil {
		t.Fatalf("expected error to clear after retry, got %v", model.rightPane.loadErr)
	}
	if got := visibleNames(model.rightPane.table.GetVisibleRows()); len(got) != 1 || got[0] != "goex-dev" {
		t.Fatalf("unexpected right pane rows after retry: %v", got)
	}
}

func teaKey(r rune) (msg tea.KeyMsg) {
	msg.Type = tea.KeyRunes
	msg.Runes = []rune{r}
	return
}
