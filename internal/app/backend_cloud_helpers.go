package app

import "strings"

func trimPrefix(value, prefix string) string {
	if prefix == "" {
		return value
	}

	if strings.HasPrefix(value, prefix) {
		return value[len(prefix):]
	}

	return value
}

func enterPrefix(fullPath, delimiter string) string {
	if fullPath != "" && !strings.HasSuffix(fullPath, delimiter) {
		return fullPath + delimiter
	}

	return fullPath
}

func parentHighlightName(prefix, rootName, delimiter string) string {
	trimmed := strings.TrimSuffix(prefix, delimiter)
	if trimmed == "" {
		return rootName
	}

	parts := strings.Split(trimmed, delimiter)
	return parts[len(parts)-1]
}

func parentPrefix(prefix, delimiter string) string {
	trimmed := strings.TrimSuffix(prefix, delimiter)
	if trimmed == "" {
		return ""
	}

	lastDelimiter := strings.LastIndex(trimmed, delimiter)
	if lastDelimiter < 0 {
		return ""
	}

	return trimmed[:lastDelimiter+len(delimiter)]
}

func hiddenBySegment(path, delimiter string) bool {
	for _, segment := range strings.Split(path, delimiter) {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}

	return false
}
