package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type externalEditSession struct {
	pane          activePane
	entryName     string
	tempPath      string
	sourceRef     TransferObjectRef
	uploadBack    bool
	initialSize   int64
	initialModUTC time.Time
}

func buildEditorCommand(path string) (*exec.Cmd, error) {
	editor := strings.TrimSpace(os.Getenv("GOEX_EDITOR"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}

	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, fmt.Errorf("editor command is empty")
	}

	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func openCommandContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = defaultLoadTimeout
	}
	return context.WithTimeout(context.Background(), timeout*4)
}

func isOpenableEntryKind(kind EntryKind) bool {
	return kind == KindObject
}

func isLocalExecutableEntry(location Location, entry Entry) bool {
	local, ok := location.(LocalLocation)
	if !ok {
		return false
	}

	path := entry.FullPath
	if path == "" {
		path = filepath.Join(local.Path, entry.Name)
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

func entrySourceRef(location Location, entry Entry) (TransferObjectRef, error) {
	objectPath := entry.FullPath
	if objectPath == "" {
		switch loc := location.(type) {
		case LocalLocation:
			objectPath = filepath.Join(loc.Path, entry.Name)
		case AzureLocation:
			objectPath = joinObjectPath(loc.Prefix, entry.Name)
		case S3Location:
			objectPath = joinObjectPath(loc.Prefix, entry.Name)
		case GCSLocation:
			objectPath = joinObjectPath(loc.Prefix, entry.Name)
		default:
			return TransferObjectRef{}, fmt.Errorf("unsupported location type: %T", location)
		}
	}

	return sourceRefForLocation(location, objectPath)
}

func (m *Model) openHighlightedInEditor() tea.Cmd {
	pane := m.activePaneRef()
	highlighted, ok := pane.highlightedEntry()
	if !ok {
		m.status = "Open: no entry selected"
		return nil
	}
	if !isOpenableEntryKind(highlighted.Kind) {
		m.status = "Open: highlight a file/object"
		return nil
	}

	return m.prepareOpenEditorCmd(m.activePane, highlighted)
}

func (m *Model) prepareOpenEditorCmd(paneID activePane, entry Entry) tea.Cmd {
	pane := m.paneByID(paneID)
	backend := pane.backend
	location := pane.location
	timeout := backend.LoadTimeout()
	commandBuilder := m.editorCommand
	if commandBuilder == nil {
		commandBuilder = buildEditorCommand
	}

	return func() tea.Msg {
		session := externalEditSession{
			pane:      paneID,
			entryName: entry.Name,
		}

		if _, ok := location.(LocalLocation); ok {
			path := entry.FullPath
			if path == "" {
				path = filepath.Join(pane.path, entry.Name)
			}
			cmd, err := commandBuilder(path)
			if err != nil {
				return editorOpenFailedMsg{err: err}
			}
			return editorPreparedMsg{session: session, cmd: cmd}
		}

		reader, ok := backend.(CopyReader)
		if !ok {
			return editorOpenFailedMsg{err: fmt.Errorf("backend does not support read streams")}
		}
		if _, ok := backend.(CopyWriter); !ok {
			return editorOpenFailedMsg{err: fmt.Errorf("backend does not support write streams")}
		}

		sourceRef, err := entrySourceRef(location, entry)
		if err != nil {
			return editorOpenFailedMsg{err: err}
		}

		ctx, cancel := openCommandContext(timeout)
		defer cancel()

		handle, err := reader.OpenCopyReader(ctx, sourceRef)
		if err != nil {
			return editorOpenFailedMsg{err: err}
		}
		defer handle.Reader.Close()

		tempFile, err := os.CreateTemp("", "goex-open-*")
		if err != nil {
			return editorOpenFailedMsg{err: err}
		}

		if _, err := io.Copy(tempFile, handle.Reader); err != nil {
			tempPath := tempFile.Name()
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
			return editorOpenFailedMsg{err: err}
		}
		if err := tempFile.Close(); err != nil {
			tempPath := tempFile.Name()
			_ = os.Remove(tempPath)
			return editorOpenFailedMsg{err: err}
		}

		info, err := os.Stat(tempFile.Name())
		if err != nil {
			tempPath := tempFile.Name()
			_ = os.Remove(tempPath)
			return editorOpenFailedMsg{err: err}
		}

		session.sourceRef = sourceRef
		session.tempPath = tempFile.Name()
		session.uploadBack = true
		session.initialSize = info.Size()
		session.initialModUTC = info.ModTime().UTC()

		cmd, err := commandBuilder(session.tempPath)
		if err != nil {
			_ = os.Remove(session.tempPath)
			return editorOpenFailedMsg{err: err}
		}

		return editorPreparedMsg{session: session, cmd: cmd}
	}
}

func (m *Model) execEditorProcessCmd(cmd *exec.Cmd, callback tea.ExecCallback) tea.Cmd {
	runner := m.execProcess
	if runner == nil {
		runner = tea.ExecProcess
	}
	return runner(cmd, callback)
}

func (m *Model) cleanupTempOpenFileCmd(session externalEditSession) tea.Cmd {
	tempPath := session.tempPath
	return func() tea.Msg {
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
		return nil
	}
}

func (m *Model) syncOpenFileChangesCmd(session externalEditSession) tea.Cmd {
	if !session.uploadBack || session.tempPath == "" {
		return func() tea.Msg {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName}
		}
	}

	pane := m.paneByID(session.pane)
	backend := pane.backend
	timeout := backend.LoadTimeout()

	return func() tea.Msg {
		defer os.Remove(session.tempPath)

		info, err := os.Stat(session.tempPath)
		if err != nil {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, err: err}
		}

		changed := info.Size() != session.initialSize || !info.ModTime().UTC().Equal(session.initialModUTC)
		if !changed {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName}
		}

		writer, ok := backend.(CopyWriter)
		if !ok {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, err: fmt.Errorf("backend does not support write streams")}
		}

		file, err := os.Open(session.tempPath)
		if err != nil {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, err: err}
		}
		defer file.Close()

		ctx, cancel := openCommandContext(timeout)
		defer cancel()

		metadata := TransferObjectMetadata{
			SizeBytes:  info.Size(),
			ModTime:    info.ModTime(),
			HasModTime: true,
		}
		target, err := writer.OpenCopyWriter(ctx, session.sourceRef, metadata)
		if err != nil {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, err: err}
		}

		if _, err := io.Copy(target, file); err != nil {
			_ = target.Close()
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, err: err}
		}
		if err := target.Close(); err != nil {
			return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, err: err}
		}

		return editorSyncDoneMsg{pane: session.pane, entry: session.entryName, uploaded: true}
	}
}

func (m *Model) enterOrOpenHighlighted() tea.Cmd {
	pane := m.activePaneRef()
	highlighted, ok := pane.highlightedEntry()
	if !ok {
		m.status = ""
		return nil
	}

	if highlighted.IsDirLike() {
		changed, err := pane.enterHighlighted(context.Background())
		if err != nil {
			m.status = err.Error()
			return nil
		}
		if changed {
			m.status = ""
			return pane.beginLoad(m.activePane)
		}
		m.status = ""
		return nil
	}

	if !isOpenableEntryKind(highlighted.Kind) {
		m.status = ""
		return nil
	}
	if isLocalExecutableEntry(pane.location, highlighted) {
		m.status = "Enter: executable file skipped (use o to open)"
		return nil
	}

	return m.prepareOpenEditorCmd(m.activePane, highlighted)
}

func (m *Model) enterHighlightedDirOnly() tea.Cmd {
	changed, err := m.activePaneRef().enterHighlighted(context.Background())
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if changed {
		m.status = ""
		return m.reloadActivePane()
	}
	m.status = ""
	return nil
}
