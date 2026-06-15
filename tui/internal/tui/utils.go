package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
)

func truncateString(s string, maxWidth int) string {
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if cur+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	return b.String()
}

func wrapLine(s string, width int) []string {
	if width < 1 {
		return []string{s}
	}
	if runewidth.StringWidth(s) <= width {
		return []string{s}
	}
	var lines []string
	remaining := s
	for runewidth.StringWidth(remaining) > width {
		cut := truncateString(remaining, width)
		lines = append(lines, cut)
		remaining = remaining[len(cut):]
	}
	if remaining != "" {
		lines = append(lines, remaining)
	}
	return lines
}

func isDetailKey(key string) bool {
	switch key {
	case "e", "d", "y":
		return true
	}
	return false
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func completePath(input string) string {
	if input == "" {
		input = "./"
	}

	dirToRead := input
	homePrefix := false
	if strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return input
		}
		dirToRead = filepath.Join(home, input[2:])
		homePrefix = true
	}

	var dir, prefix string
	if strings.HasSuffix(dirToRead, "/") {
		dir = dirToRead
		prefix = ""
	} else {
		dir = filepath.Dir(dirToRead)
		prefix = filepath.Base(dirToRead)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return input
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if e.IsDir() {
			matches = append(matches, name+"/")
		} else {
			matches = append(matches, name)
		}
	}

	if len(matches) == 0 {
		return input
	}

	sort.Strings(matches)
	cp := commonPrefix(matches)

	var result string
	if strings.HasSuffix(input, "/") {
		result = input + cp
	} else {
		result = input[:len(input)-len(prefix)] + cp
	}

	if homePrefix && strings.HasPrefix(result, "~/") {
		home, _ := os.UserHomeDir()
		fullPath := filepath.Join(home, result[2:])
		if len(matches) == 1 && !strings.HasSuffix(result, "/") {
			info, err := os.Stat(fullPath)
			if err == nil && info.IsDir() {
				result += "/"
			}
		}
	} else if len(matches) == 1 && !strings.HasSuffix(result, "/") {
		info, err := os.Stat(result)
		if err == nil && info.IsDir() {
			result += "/"
		}
	}

	return result
}

func commonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	p := strs[0]
	for _, s := range strs[1:] {
		for i := 0; i < len(p) && i < len(s); i++ {
			if p[i] != s[i] {
				p = p[:i]
				break
			}
		}
		if len(s) < len(p) {
			p = s
		}
	}
	return p
}
