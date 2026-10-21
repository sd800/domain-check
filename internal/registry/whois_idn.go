package registry

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// Accept an alias only when the registry explicitly describes the same bundle.
// These checks never fetch another name or follow a referral.
func markWHOISBundle(domain string, lines []whoisLine, profile whoisProfile) error {
	switch {
	case profile.has(cnnicVariantNames):
		var headers []int
		matched := false
		start := 0
		for start < len(lines) && (strings.TrimSpace(lines[start].raw) == "" || strings.HasPrefix(strings.TrimSpace(lines[start].raw), "%")) {
			start++
		}
		for i := start; i+1 < len(lines); i += 2 {
			if lines[i].key != "domain name" || lines[i+1].key != "puny name" {
				break
			}
			name, err := NormalizeDomain(lines[i].value)
			puny, punyErr := NormalizeDomain(lines[i+1].value)
			if err != nil || punyErr != nil || name != puny || (!strings.HasSuffix(name, ".xn--fiqs8s") && !strings.HasSuffix(name, ".xn--fiqz9s")) {
				return queryError("invalid-response", "inconsistent WHOIS domain aliases")
			}
			headers = append(headers, i)
			matched = matched || name == domain
		}
		if matched && len(headers) > 1 {
			for _, i := range headers {
				lines[i].alias = true
			}
		}
	case profile.has(hkBundledNames):
		for i := range lines {
			if lines[i].key == "domain name" && hkWHOISNameMatches(domain, lines[i].value) {
				lines[i].alias = true
				return nil
			}
		}
	case profile.has(twnicVariantNames):
		return markTWNICBundle(domain, lines)
	}
	return nil
}

func hkWHOISNameMatches(domain, value string) bool {
	primary, bundled := value, ""
	if index := strings.Index(strings.ToLower(value), "bundled domain name:"); index >= 0 {
		primary, bundled = value[:index], value[index+len("bundled domain name:"):]
	}
	parts := strings.Fields(primary)
	if len(parts) == 0 {
		return false
	}
	// HKIRC uses both NAME.HK/.香港 and NAME.HK /.香港.
	compact := strings.Split(strings.Join(parts, ""), "/.")
	name, err := NormalizeDomain(compact[0])
	if err != nil || len(parts) > 1 && !strings.HasPrefix(parts[1], "/.") {
		return false
	}
	names := []string{name}
	for _, suffix := range compact[1:] {
		if strings.ContainsAny(suffix, "/ :") {
			return false
		}
		alias, err := NormalizeDomain(name[:strings.LastIndexByte(name, '.')+1] + suffix)
		if err != nil {
			return false
		}
		names = append(names, alias)
	}
	if bundled != "" {
		for _, candidate := range strings.Fields(bundled) {
			alias, err := NormalizeDomain(candidate)
			if err != nil {
				return false
			}
			names = append(names, alias)
		}
	}
	for _, name := range names {
		if name == domain {
			return true
		}
	}
	return false
}

func twnicName(value string) (string, bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.HasPrefix(fields[1], "(") || !strings.HasSuffix(fields[1], ")") {
		return "", false
	}
	name, err := NormalizeDomain(fields[0])
	puny, punyErr := NormalizeDomain(strings.Trim(fields[1], "()"))
	return name, err == nil && punyErr == nil && name == puny
}

func markTWNICBundle(domain string, lines []whoisLine) error {
	primaryIndex := -1
	var primary string
	matched := false
	for i, line := range lines {
		switch line.key {
		case "註冊原型域名", "繁體對照字域名", "簡体對照字域名", "簡體對照字域名", "自動解析域名":
			name, valid := twnicName(line.value)
			if !valid {
				return queryError("invalid-response", "inconsistent WHOIS domain aliases")
			}
			if line.key == "註冊原型域名" {
				primary, primaryIndex = name, i
			}
			matched = matched || name == domain
		}
	}
	if primaryIndex < 0 {
		return nil
	}
	if !matched {
		matched = twnicVariantMatches(domain, primary, lines)
	}
	if matched {
		lines[primaryIndex].alias = true
	}
	return nil
}

func twnicVariantMatches(domain, primary string, lines []whoisLine) bool {
	query := strings.Split(domain, ".")
	base := strings.Split(primary, ".")
	if len(query) != 2 || len(base) != 2 || query[1] != "xn--kprw13d" || (base[1] != "xn--kpry57d" && base[1] != "xn--kprw13d") {
		return false
	}
	queryName, err := idna.Lookup.ToUnicode(query[0])
	baseName, baseErr := idna.Lookup.ToUnicode(base[0])
	if err != nil || baseErr != nil {
		return false
	}
	q, b := []rune(queryName), []rune(baseName)
	if len(q) != len(b) {
		return false
	}
	inside, position := false, 0
	for _, line := range lines {
		if strings.HasPrefix(line.key, "保留字域名(") || strings.HasPrefix(line.key, "保留字域名（") {
			inside = true
			continue
		}
		if !inside || strings.TrimSpace(line.raw) == "" {
			continue
		}
		if line.labeled {
			break
		}
		if position >= len(q) {
			return false
		}
		queryFound, baseFound := false, false
		for _, character := range strings.Fields(line.value) {
			if utf8.RuneCountInString(character) != 1 {
				return false
			}
			r, _ := utf8.DecodeRuneInString(character)
			queryFound = queryFound || r == q[position]
			baseFound = baseFound || r == b[position]
		}
		if !queryFound || !baseFound {
			return false
		}
		position++
	}
	return position > 0 && position == len(q)
}

func normalizeBundledDNSSEC(value string) string {
	parts := strings.Split(value, "/")
	normalized := ""
	for _, part := range parts {
		part = normalizeWHOISDNSSEC(strings.TrimSpace(part))
		if part != "unsigned" && part != "signedDelegation" {
			return value
		}
		if normalized != "" && normalized != part {
			return value
		}
		normalized = part
	}
	return normalized
}
