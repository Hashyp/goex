package app

import (
	"slices"
	"strings"
	"time"
)

type EntryKind int

const (
	KindContainer EntryKind = iota
	KindDirectory
	KindBlob
)

type Entry struct {
	ID         string
	Name       string
	FullPath   string
	Kind       EntryKind
	SizeBytes  int64
	ModTime    time.Time
	HasModTime bool
}

func (e Entry) IsDirLike() bool {
	return e.Kind == KindContainer || e.Kind == KindDirectory
}

func (e Entry) TypeOrSize() string {
	switch e.Kind {
	case KindContainer:
		return "<CNT>"
	case KindDirectory:
		return "<DIR>"
	case KindBlob:
		return formatSize(e.SizeBytes)
	default:
		return ""
	}
}

func sortEntries(entries []Entry) {
	slices.SortFunc(entries, func(a, b Entry) int {
		aDir := a.IsDirLike()
		bDir := b.IsDirLike()
		if aDir && !bDir {
			return -1
		}
		if !aDir && bDir {
			return 1
		}

		aName := strings.ToLower(a.Name)
		bName := strings.ToLower(b.Name)
		switch {
		case aName < bName:
			return -1
		case aName > bName:
			return 1
		default:
			return 0
		}
	})
}
