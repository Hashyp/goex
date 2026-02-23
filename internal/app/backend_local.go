package app

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
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

func (b LocalBackend) Delete(_ context.Context, state Location, highlighted Entry) error {
	if !isDeleteTargetKind(highlighted.Kind) {
		return nil
	}

	local, ok := state.(LocalLocation)
	if !ok {
		return ErrInvalidLocation
	}

	target := filepath.Join(local.Path, highlighted.Name)
	if highlighted.Kind == KindDirectory {
		return b.fs.RemoveAll(target)
	}

	return b.fs.Remove(target)
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

func (b LocalBackend) EnumerateCopy(_ context.Context, state Location, selected []Entry, destination Location) ([]TransferPlanItem, error) {
	local, ok := state.(LocalLocation)
	if !ok {
		return nil, ErrInvalidLocation
	}

	plan := make([]TransferPlanItem, 0, len(selected))
	for _, entry := range selected {
		switch entry.Kind {
		case KindObject:
			sourcePath := entry.FullPath
			if sourcePath == "" {
				sourcePath = filepath.Join(local.Path, entry.Name)
			}

			srcRef, err := sourceRefForLocation(local, sourcePath)
			if err != nil {
				return nil, err
			}
			dstRef, err := resolveDestinationRef(destination, entry.Name)
			if err != nil {
				return nil, err
			}
			plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
		case KindDirectory:
			dirPath := entry.FullPath
			if dirPath == "" {
				dirPath = filepath.Join(local.Path, entry.Name)
			}

			if err := filepath.WalkDir(dirPath, func(filePath string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() {
					return nil
				}

				relativeToDir, err := filepath.Rel(dirPath, filePath)
				if err != nil {
					return err
				}
				relative := filepath.ToSlash(filepath.Join(entry.Name, relativeToDir))

				srcRef, err := sourceRefForLocation(local, filePath)
				if err != nil {
					return err
				}
				dstRef, err := resolveDestinationRef(destination, relative)
				if err != nil {
					return err
				}

				plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
				return nil
			}); err != nil {
				return nil, err
			}
		}
	}

	return plan, nil
}

func (b LocalBackend) OpenCopyReader(_ context.Context, source TransferObjectRef) (CopyReadHandle, error) {
	file, err := os.Open(source.Path)
	if err != nil {
		return CopyReadHandle{}, err
	}

	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return CopyReadHandle{}, err
	}

	return CopyReadHandle{
		Reader: file,
		Metadata: TransferObjectMetadata{
			SizeBytes:  stat.Size(),
			ModTime:    stat.ModTime(),
			HasModTime: true,
		},
	}, nil
}

func (b LocalBackend) CopyDestinationExists(_ context.Context, destination TransferObjectRef) (bool, error) {
	_, err := os.Stat(destination.Path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func (b LocalBackend) OpenCopyWriter(_ context.Context, destination TransferObjectRef, _ TransferObjectMetadata) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(destination.Path), 0o755); err != nil {
		return nil, err
	}

	return os.Create(destination.Path)
}

var ErrInvalidLocation = errors.New("invalid location type for backend")
