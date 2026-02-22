package app

import (
	"fmt"
)

const (
	columnKeyName    = "name"
	columnKeyNameRaw = "__name_raw"
	columnKeyEntryID = "__entry_id"
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
