package tui

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

type totpParams struct {
	secret    []byte
	digits    int
	period    int
	algorithm string
	steam     bool
}

type TOTPDetailView struct {
	value       string
	code        string
	remaining   int
	period      int
	digits      int
	timestamp   uint64
	copySuccess bool
	width       int
	height      int
}

func (v *TOTPDetailView) SetValue(s string) {
	v.value = s
}

func (v *TOTPDetailView) SetCode(code string) {
	v.code = code
}

func (v *TOTPDetailView) SetRemaining(r int) {
	v.remaining = r
}

func (v *TOTPDetailView) SetPeriod(p int) {
	v.period = p
}

func (v *TOTPDetailView) SetDigits(d int) {
	v.digits = d
}

func (v *TOTPDetailView) SetTimestamp(ts uint64) {
	v.timestamp = ts
}

func (v *TOTPDetailView) SetCopySuccess(b bool) {
	v.copySuccess = b
}

func (v *TOTPDetailView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *TOTPDetailView) Code() string {
	return v.code
}

func (v *TOTPDetailView) Remaining() int {
	return v.remaining
}

func (v *TOTPDetailView) Period() int {
	return v.period
}

func (v *TOTPDetailView) ComputeCode() {
	raw := v.value
	params, err := parseOtpAuthURI(raw)
	if err != nil {
		secret := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
		secret = strings.TrimRight(secret, "=")
		key, decErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
		if decErr != nil {
			v.code = "invalid secret"
			v.remaining = 0
			return
		}
		params = totpParams{secret: key, digits: 6, period: 30, algorithm: "SHA1"}
	}

	v.digits = params.digits
	v.period = params.period

	ts := time.Now().Unix() / int64(params.period)
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(ts))

	h := hmac.New(newHash(params.algorithm), params.secret)
	h.Write(counter)
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	fullCode := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF

	if params.steam {
		const steamChars = "23456789BCDFGHJKMNPQRTVWXY"
		code := fullCode
		var result strings.Builder
		for i := 0; i < 5; i++ {
			result.WriteByte(steamChars[code%uint32(len(steamChars))])
			code /= uint32(len(steamChars))
		}
		v.code = result.String()
	} else {
		divisor := uint32(1)
		for i := 0; i < params.digits; i++ {
			divisor *= 10
		}
		code := fullCode % divisor
		fmtStr := fmt.Sprintf("%%0%dd", params.digits)
		v.code = fmt.Sprintf(fmtStr, code)
	}

	v.remaining = int(params.period - int(time.Now().Unix()%int64(params.period)))
}

func (v *TOTPDetailView) View() string {
	width := v.width
	if width < 1 {
		width = 30
	}

	var b strings.Builder

	codeStyle := totpCodeStyle.Copy().Width(width - 4).Align(lipgloss.Center)
	b.WriteString(codeStyle.Render(v.code))
	b.WriteString("\n\n")

	barWidth := width - 8
	if barWidth < 10 {
		barWidth = 10
	}
	period := v.period
	if period < 1 {
		period = 30
	}
	filled := (v.remaining * barWidth) / period
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	b.WriteString(lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).Render(bar))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).Render(fmt.Sprintf("%ds", v.remaining)))
	b.WriteString("\n\n")

	maxWidth := width - 4
	if maxWidth < 10 {
		maxWidth = 10
	}
	for _, line := range wrapLine(v.value, maxWidth) {
		b.WriteString(menuStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
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
		} else if val, e := strconv.Atoi(d); e == nil && val > 0 {
			p.digits = val
		}
	}

	if pr := q.Get("period"); pr != "" {
		if val, e := strconv.Atoi(pr); e == nil && val > 0 {
			p.period = val
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
