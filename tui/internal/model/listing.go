package model

import (
	"strings"
)

type ListItem struct {
	Name     string
	FullPath string
	IsEntry  bool
	IsAttr   bool
}

func normalizePathPrefix(prefix string) string {
	if prefix == "" || prefix == "/" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")
	return prefix
}

func EntryPath(groupPath, entryID string) string {
	groupPath = normalizePathPrefix(groupPath)
	if groupPath == "" {
		return "/" + entryID
	}
	return groupPath + "/" + entryID
}

func ParentPath(path string) string {
	path = normalizePathPrefix(path)
	if path == "" {
		return ""
	}
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash <= 0 {
		return ""
	}
	return path[:lastSlash]
}
