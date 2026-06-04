package model

import (
	"sort"
	"strings"

	"github.com/tapass/tapass-tools/vault"
)

type ListItem struct {
	Name     string
	FullPath string
	IsEntry  bool
	IsAttr   bool
}

func ListChildren(entries map[string]vault.Entry, prefix string) []ListItem {
	resultMap := make(map[string]ListItem)

	prefix = normalizePathPrefix(prefix)

	for key := range entries {
		if !strings.HasPrefix(key, prefix+"/") {
			continue
		}

		rest := strings.TrimPrefix(key, prefix+"/")
		if rest == "" {
			continue
		}

		parts := strings.SplitN(rest, "/", 2)
		name := parts[0]
		fullPath := prefix + "/" + name

		isEntry := false
		if len(parts) == 2 && !strings.Contains(parts[1], "/") {
			isEntry = true
		} else if len(parts) == 1 {
			continue
		}

		existing, exists := resultMap[fullPath]
		if !exists || isEntry {
			resultMap[fullPath] = ListItem{
				Name:     name,
				FullPath: fullPath,
				IsEntry:  isEntry,
			}
		}
		if exists && !existing.IsEntry && isEntry {
			item := resultMap[fullPath]
			item.IsEntry = true
			resultMap[fullPath] = item
		}
	}

	for key := range entries {
		if !strings.HasPrefix(key, prefix+"/") {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+"/")
		if rest == "" {
			continue
		}
		parts := strings.SplitN(rest, "/", 2)
		name := parts[0]
		fullPath := prefix + "/" + name

		if len(parts) == 2 && strings.Contains(parts[1], "/") {
			if item, ok := resultMap[fullPath]; ok && !item.IsEntry {
				continue
			}
		}
	}

	result := make([]ListItem, 0, len(resultMap))
	for _, item := range resultMap {
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsEntry != result[j].IsEntry {
			return !result[i].IsEntry
		}
		return result[i].Name < result[j].Name
	})

	return result
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

func GetEntryAttrs(entries map[string]vault.Entry, entryPath string) map[string][]byte {
	attrs := make(map[string][]byte)
	entryPath = normalizePathPrefix(entryPath)

	for key, e := range entries {
		if !strings.HasPrefix(key, entryPath+"/") {
			continue
		}
		attrName := strings.TrimPrefix(key, entryPath+"/")
		if strings.Contains(attrName, "/") {
			continue
		}
		if attrName == "" {
			continue
		}
		attrs[attrName] = e.Value
	}

	return attrs
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

func IsEntryPath(entries map[string]vault.Entry, path string) bool {
	path = normalizePathPrefix(path)
	for key := range entries {
		if !strings.HasPrefix(key, path+"/") {
			continue
		}
		rest := strings.TrimPrefix(key, path+"/")
		if rest != "" && !strings.Contains(rest, "/") {
			return true
		}
	}
	return false
}

func GetAllEntryPaths(entries map[string]vault.Entry) []string {
	entryMap := make(map[string]bool)

	for key := range entries {
		parts := strings.Split(strings.TrimPrefix(key, "/"), "/")
		if len(parts) >= 2 {
			entryPath := "/" + strings.Join(parts[:len(parts)-1], "/")
			entryMap[entryPath] = true
		}
	}

	result := make([]string, 0, len(entryMap))
	for path := range entryMap {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}
