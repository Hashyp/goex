package app

import (
	"fmt"

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

func getDirAndFiles(fs FileSystem, path string) ([]table.Row, error) {
	entries, err := fs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	rows := []table.Row{}
	for _, entry := range entries {
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
