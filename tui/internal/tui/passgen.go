package tui

import (
	"crypto/rand"
	"math/big"
)

const (
	charUppercase  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charLowercase  = "abcdefghijklmnopqrstuvwxyz"
	charDigits     = "0123456789"
	charSymbols    = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	ambiguousChars = "0Oo1lI"
)

type PassGenRules struct {
	Length           int
	Uppercase        bool
	Lowercase        bool
	Digits           bool
	Symbols          bool
	ExcludeAmbiguous bool
}

func defaultPassGenRules() PassGenRules {
	return PassGenRules{
		Length:           20,
		Uppercase:        true,
		Lowercase:        true,
		Digits:           true,
		Symbols:          true,
		ExcludeAmbiguous: true,
	}
}

type PassGenState struct {
	rules     PassGenRules
	generated string
	cursor    int
	width     int
	height    int
}

func newPassGenState() PassGenState {
	return PassGenState{
		rules: defaultPassGenRules(),
	}
}

const (
	pgCursorLength = iota
	pgCursorUppercase
	pgCursorLowercase
	pgCursorDigits
	pgCursorSymbols
	pgCursorExcludeAmbiguous
	pgCursorCount
)

func (s *PassGenState) moveCursor(delta int) {
	s.cursor = (s.cursor + delta) % pgCursorCount
	if s.cursor < 0 {
		s.cursor += pgCursorCount
	}
}

func (s *PassGenState) toggleBool() {
	switch s.cursor {
	case pgCursorUppercase:
		s.rules.Uppercase = !s.rules.Uppercase
	case pgCursorLowercase:
		s.rules.Lowercase = !s.rules.Lowercase
	case pgCursorDigits:
		s.rules.Digits = !s.rules.Digits
	case pgCursorSymbols:
		s.rules.Symbols = !s.rules.Symbols
	case pgCursorExcludeAmbiguous:
		s.rules.ExcludeAmbiguous = !s.rules.ExcludeAmbiguous
	}
}

func (s *PassGenState) adjustLength(delta int) {
	s.rules.Length += delta
	if s.rules.Length < 4 {
		s.rules.Length = 4
	}
	if s.rules.Length > 128 {
		s.rules.Length = 128
	}
}

func (s *PassGenState) generate() {
	charset := s.buildCharset()
	if len(charset) == 0 {
		s.generated = ""
		return
	}

	var required [][]byte
	if s.rules.Uppercase {
		required = append(required, []byte(s.filterCharset(charUppercase)))
	}
	if s.rules.Lowercase {
		required = append(required, []byte(s.filterCharset(charLowercase)))
	}
	if s.rules.Digits {
		required = append(required, []byte(s.filterCharset(charDigits)))
	}
	if s.rules.Symbols {
		required = append(required, []byte(s.filterCharset(charSymbols)))
	}

	result := make([]byte, s.rules.Length)
	used := make(map[int]bool)

	for i, req := range required {
		if len(req) == 0 {
			continue
		}
		pos := s.randInt(s.rules.Length - len(required) + i + 1)
		for used[pos] {
			pos++
		}
		used[pos] = true
		result[pos] = req[s.randInt(len(req))]
	}

	for i := 0; i < s.rules.Length; i++ {
		if used[i] {
			continue
		}
		result[i] = charset[s.randInt(len(charset))]
	}

	s.generated = string(result)
}

func (s *PassGenState) buildCharset() []byte {
	var cs []byte
	if s.rules.Uppercase {
		cs = append(cs, s.filterCharset(charUppercase)...)
	}
	if s.rules.Lowercase {
		cs = append(cs, s.filterCharset(charLowercase)...)
	}
	if s.rules.Digits {
		cs = append(cs, s.filterCharset(charDigits)...)
	}
	if s.rules.Symbols {
		cs = append(cs, s.filterCharset(charSymbols)...)
	}
	return cs
}

func (s *PassGenState) filterCharset(chars string) []byte {
	if !s.rules.ExcludeAmbiguous {
		return []byte(chars)
	}
	ambiguous := make(map[rune]bool)
	for _, r := range ambiguousChars {
		ambiguous[r] = true
	}
	var filtered []byte
	for _, r := range chars {
		if !ambiguous[r] {
			filtered = append(filtered, byte(r))
		}
	}
	return filtered
}

func (s *PassGenState) randInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}
