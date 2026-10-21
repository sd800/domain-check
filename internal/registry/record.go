// Package registry queries registry services, with DNS nameserver fallback for .edu.cn.
package registry

import (
	"fmt"
	"net"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/idna"
)

type Result string

const (
	NoRecord      Result = "no-record"
	NotRegistered Result = "not-registered"
	Reserved      Result = "reserved"
	NotAllowed    Result = "not-allowed"
	Restricted    Result = "registration-restricted"
	Unavailable   Result = "not-available"
)

type Record struct {
	Domain          string
	Registration    string
	Expiration      string
	Changed         string
	Statuses        []string
	Registrar       string
	Registrant      string
	RegistrantEmail []string
	Nameservers     []string
	DNSSEC          string
	Result          Result
}

// QueryError describes a failed query, never an availability determination.
type QueryError struct {
	Code   string
	Detail string
}

func (e *QueryError) Error() string { return e.Code + ": " + e.Detail }

func queryError(code, detail string) error {
	return &QueryError{Code: code, Detail: Clean(detail)}
}

func NormalizeDomain(input string) (string, error) {
	input = strings.TrimSpace(input)
	ascii, err := idna.Lookup.ToASCII(input)
	if err != nil {
		return "", fmt.Errorf("invalid domain: %w", err)
	}
	ascii = strings.ToLower(strings.TrimSuffix(ascii, "."))
	if len(ascii) > 253 || !strings.Contains(ascii, ".") || net.ParseIP(ascii) != nil {
		return "", fmt.Errorf("expected a domain such as example.com")
	}
	for _, label := range strings.Split(ascii, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("invalid domain label")
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
				return "", fmt.Errorf("invalid domain character")
			}
		}
	}
	tld := ascii[strings.LastIndexByte(ascii, '.')+1:]
	if strings.Trim(tld, "0123456789") == "" {
		return "", fmt.Errorf("invalid domain suffix")
	}
	return ascii, nil
}

// Clean prevents registry-supplied control characters from operating the terminal.
func Clean(s string) string {
	// Most labels and values already have ordinary single spaces. Preserve
	// their backing string instead of allocating a field slice and a copy.
	space := false
	for i, r := range s {
		separator := r <= ' ' || r == 127
		if r >= 128 {
			separator = unicode.IsSpace(r) || unicode.IsControl(r) || unicode.In(r, unicode.Cf)
		}
		if separator && (r != ' ' || i == 0 || space) {
			return cleanFields(s)
		}
		space = separator
	}
	if space {
		return cleanFields(s)
	}
	return s
}

func cleanFields(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.In(r, unicode.Cf)
	}), " ")
}

func useful(s string) string {
	s = Clean(s)
	lower := strings.ToLower(s)
	if redacted := strings.Trim(lower, "()[] "); redacted == "redacted" || strings.HasPrefix(redacted, "redacted for ") {
		return ""
	}
	switch lower {
	case "", "-", "--", "n/a", "null", "none", "not available", "not disclosed", "redacted", "redacted for privacy", "data protected":
		return ""
	}
	for _, prefix := range []string{"data redacted", "redacted for ", "not disclosed", "hidden upon user request", "please visit ", "please query ", "visit www."} {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	return s
}

func appendUnique(items []string, value string) []string {
	value = useful(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if strings.EqualFold(item, value) {
			return items
		}
	}
	return append(items, value)
}

// Zoned timestamps become UTC and are labeled. Dates and unzoned timestamps
// retain their precision and wall time while using a consistent numeric format.
func formatTime(s string) string {
	s = useful(s)
	for _, layout := range []string{
		time.RFC3339Nano, "2006-01-02T15:04:05-0700", "2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05 -0700", "2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05 MST", "02-Jan-2006 15:04:05 MST",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			// time.Parse accepts unknown zone abbreviations with a fabricated offset.
			if strings.HasSuffix(layout, "MST") && !strings.HasSuffix(s, "UTC") && !strings.HasSuffix(s, "GMT") {
				continue
			}
			return t.UTC().Format("2006-01-02 15:04:05") + " UTC"
		}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05", "2006-01-02T15:04:05",
		"02-Jan-2006 15:04:05", "2 Jan 2006 15:04:05",
		"2006/01/02 15:04:05", "2006.01.02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	for _, layout := range []string{
		"2006-01-02", "2006-1-2", "2-Jan-2006", "2 Jan 2006",
		"2006-Jan-2", "January 2, 2006", "2006/1/2", "2006.1.2",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}
