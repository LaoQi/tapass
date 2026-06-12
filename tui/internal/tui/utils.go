package tui

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"fmt"
	"hash"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

func parseOtpAuthURI(raw string) (totpParams, error) {
	p := totpParams{digits: 6, period: 30, algorithm: "SHA1"}

	if !strings.HasPrefix(raw, "otpauth://totp/") {
		return p, fmt.Errorf("not an otpauth://totp/ URI")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return p, fmt.Errorf("parse URI: %w", err)
	}

	q := u.Query()

	secretStr := q.Get("secret")
	if secretStr == "" {
		return p, fmt.Errorf("missing secret parameter")
	}
	secretUpper := strings.ToUpper(strings.ReplaceAll(secretStr, " ", ""))
	p.secret, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(secretUpper, "="))
	if err != nil {
		return p, fmt.Errorf("decode secret: %w", err)
	}

	if d := q.Get("digits"); d != "" {
		if strings.EqualFold(d, "S") {
			p.steam = true
			p.digits = 5
		} else if v, err := strconv.Atoi(d); err == nil && v > 0 {
			p.digits = v
		}
	}

	if pr := q.Get("period"); pr != "" {
		if v, err := strconv.Atoi(pr); err == nil && v > 0 {
			p.period = v
		}
	}

	if alg := q.Get("algorithm"); alg != "" {
		p.algorithm = strings.ToUpper(alg)
	}

	return p, nil
}

func newHash(algorithm string) func() hash.Hash {
	switch algorithm {
	case "SHA256":
		return sha256.New
	case "SHA512":
		return sha512.New
	default:
		return sha1.New
	}
}
