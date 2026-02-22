package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type LocalBackend struct {
	fs        FileSystem
	startPath string
}

func NewLocalBackend(fs FileSystem, startPath string) LocalBackend {
	return LocalBackend{fs: fs, startPath: startPath}
}

func (b LocalBackend) InitialLocation() Location {
	return LocalLocation{Path: b.startPath}
}

func (b LocalBackend) DisplayPath(state Location) string {
	local, ok := state.(LocalLocation)
	if !ok {
		return "<invalid-local-location>"
	}

	return local.Path
}

func (b LocalBackend) ParentHighlightName(state Location) string {
	local, ok := state.(LocalLocation)
	if !ok {
		return ""
	}

	base := filepath.Base(local.Path)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}

	return base
}

func (b LocalBackend) LoadTimeout() time.Duration {
	return 10 * time.Second
}

func (b LocalBackend) List(_ context.Context, state Location, showHidden bool) ([]Entry, error) {
	local, ok := state.(LocalLocation)
	if !ok {
		return nil, ErrInvalidLocation
	}

	entries, err := b.fs.ReadDir(local.Path)
	if err != nil {
		return nil, err
	}

	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if !showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		kind := KindObject
		if info.IsDir() {
			kind = KindDirectory
		}

		fullPath := filepath.Join(local.Path, entry.Name())
		items = append(items, Entry{
			ID:         fullPath,
			Name:       entry.Name(),
			FullPath:   fullPath,
			Kind:       kind,
			SizeBytes:  info.Size(),
			ModTime:    info.ModTime(),
			HasModTime: true,
		})
	}

	sortEntries(items)
	return items, nil
}

func (b LocalBackend) Enter(_ context.Context, state Location, highlighted Entry) (Location, bool, error) {
	if !highlighted.IsDirLike() {
		return state, false, nil
	}

	local, ok := state.(LocalLocation)
	if !ok {
		return state, false, ErrInvalidLocation
	}

	target := filepath.Join(local.Path, highlighted.Name)
	info, err := b.fs.Stat(target)
	if err != nil {
		return state, false, err
	}
	if !info.IsDir() {
		return state, false, nil
	}

	return LocalLocation{Path: target}, true, nil
}

func (b LocalBackend) Parent(state Location) (Location, bool) {
	local, ok := state.(LocalLocation)
	if !ok {
		return state, false
	}

	parent := filepath.Dir(local.Path)
	if parent == local.Path {
		return state, false
	}

	return LocalLocation{Path: parent}, true
}

var ErrInvalidLocation = errors.New("invalid location type for backend")
