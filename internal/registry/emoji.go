package registry

import (
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
	"golang.org/x/net/idna"
)

var emojiIDNA = idna.New(idna.MapForLookup(), idna.CheckJoiners(false))

func domainASCII(name string) (string, error) {
	ascii, err := idna.Lookup.ToASCII(name)
	if err == nil {
		return ascii, nil
	}
	// Some registries support emoji joined with U+200D. Relax ContextJ only
	// for complete pictographic grapheme clusters, never arbitrary joiners.
	candidate, candidateErr := emojiIDNA.ToASCII(name)
	if candidateErr == nil {
		decoded, decodeErr := emojiIDNA.ToUnicode(candidate)
		if decodeErr == nil && emojiJoiners(decoded) {
			return candidate, nil
		}
	}
	return "", err
}

func emojiJoiners(name string) bool {
	found := false
	for _, r := range name {
		if unicode.In(r, unicode.Cf) && r != '\u200d' {
			return false
		}
	}
	clusters := uniseg.NewGraphemes(name)
	for clusters.Next() {
		cluster := clusters.Str()
		if !strings.ContainsRune(cluster, '\u200d') {
			continue
		}
		found = true
		// UAX #29 groups pictographs across ZWJ. Dangling joiners,
		// letter joins and empty/zero-width components fail this check.
		for _, part := range strings.Split(cluster, "\u200d") {
			if part == "" || uniseg.StringWidth(part) == 0 {
				return false
			}
		}
	}
	return found
}

// UnicodeDomain returns a locally decoded display name only after validating
// its ASCII identity and round trip. It never queries another service.
func UnicodeDomain(domain string) string {
	ascii, err := normalizeQueryIdentity(domain)
	if err != nil {
		return ""
	}
	name, err := idna.Lookup.ToUnicode(ascii)
	if err != nil {
		name, err = emojiIDNA.ToUnicode(ascii)
		if err != nil || !emojiJoiners(name) {
			return ""
		}
	}
	if name == ascii {
		return ""
	}
	if roundTrip, err := normalizeQueryIdentity(name); err != nil || roundTrip != ascii {
		return ""
	}
	// Preserve emoji ZWJ only in this validated name. Other formatting and
	// terminal controls must never be introduced by the display conversion.
	for _, r := range name {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.In(r, unicode.Cf) && r != '\u200d' {
			return ""
		}
	}
	return name
}
