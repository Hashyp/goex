package app

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/evertras/bubble-table/table"
)

const (
	columnKeyName    = "name"
	columnKeyNameRaw = "__name_raw"
	columnKeySize    = "size"
	columnKeyDate    = "date"
	columnKeyTime    = "time"
)

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

func getDirAndFiles(fs FileSystem, path string, showHidden bool) ([]table.Row, error) {
	entries, err := fs.ReadDir(path)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		if a.IsDir() && !b.IsDir() {
			return -1
		}
		if !a.IsDir() && b.IsDir() {
			return 1
		}

		switch {
		case a.Name() < b.Name():
			return -1
		case a.Name() > b.Name():
			return 1
		default:
			return 0
		}
	})

	rows := []table.Row{}
	for _, entry := range entries {
		if !showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime()
		var size string
		if info.IsDir() {
			size = "<DIR>"
		} else {
			size = formatSize(info.Size())
		}

		rows = append(rows, table.NewRow(table.RowData{
			columnKeyName:    entry.Name(),
			columnKeyNameRaw: entry.Name(),
			columnKeySize:    size,
			columnKeyDate:    modTime.Format("2006-01-02"),
			columnKeyTime:    modTime.Format("15:04:05"),
		}))
	}

	return rows, nil
}
