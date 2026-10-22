package registry

import (
	"regexp"
	"strings"
)

var kyrgyzDomainHeader = regexp.MustCompile(`^Domain (\S+)(?: \(([A-Z][A-Z -]*)\))?$`)

func tokenizeKyrgyzHeader(lines []whoisLine) []whoisLine {
	for i, line := range lines {
		match := kyrgyzDomainHeader.FindStringSubmatch(strings.TrimSpace(line.raw))
		if match == nil {
			continue
		}
		lines[i] = whoisLine{raw: line.raw, key: "domain name", value: match[1], labeled: true}
		if match[2] == "" {
			break
		}
		lines = append(lines, whoisLine{})
		copy(lines[i+2:], lines[i+1:])
		lines[i+1] = whoisLine{raw: "Domain Status: " + match[2], key: "domain status", value: match[2], labeled: true}
		break
	}
	return lines
}

func koreanWHOISResult(domain string, lines []whoisLine) Result {
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	if tld != "kr" && tld != "xn--3e0b707e" {
		return ""
	}
	queried := false
	var result Result
	for _, line := range lines {
		if whoisKeys[line.key] == "domain" {
			return ""
		}
		if line.key == "query" {
			name, err := NormalizeDomain(line.value)
			if err != nil || name != domain {
				return ""
			}
			queried = true
		}
		switch Clean(line.raw) {
		case "This request domain name is restricted to specifically qualified registrants for stable management of domain names and public interest.", "등록자격이 제한된 도메인이름입니다.", "제한되어 있습니다.":
			if result != "" && result != EligibilityRestricted {
				return ""
			}
			result = EligibilityRestricted
		case "상기 도메인이름은 등록되어 있지 않습니다.", "상기 호스트이름은 등록되어 있지 않습니다.":
			if result != "" && result != NoRecord {
				return ""
			}
			result = NoRecord
		}
	}
	if queried {
		return result
	}
	return ""
}

// Nominet may describe a delegated namespace instead of returning an object.
// Match the queried identity and exact notices before interpreting that reply.
func nominetWHOISResult(domain string, lines []whoisLine) Result {
	if !strings.HasSuffix(domain, ".uk") {
		return ""
	}
	var body []string
	for _, line := range lines {
		text := strings.TrimSpace(line.raw)
		if strings.HasPrefix(text, "WHOIS lookup made at ") {
			break
		}
		if whoisKeys[line.key] == "domain" {
			return ""
		}
		if text != "" {
			body = append(body, text)
		}
	}
	if len(body) == 3 && body[0] == "The domains, and sub-domains, of mod.uk and mil.uk are for use by the" && body[1] == "UK Armed Forces and Ministry of Defence establishments and systems." && body[2] == "Correspondence relating to them should be sent to hostmaster@mod.gov.uk." {
		if domain == "mod.uk" || domain == "mil.uk" || strings.HasSuffix(domain, ".mod.uk") || strings.HasSuffix(domain, ".mil.uk") {
			return NotProvided
		}
	}
	if len(body) < 2 || body[0] != `Error for "`+domain+`".` {
		return ""
	}
	if len(body) == 2 && body[1] == "Nominet is not the registry for this domain name." {
		return NotProvided
	}
	if len(body) == 4 && body[1] == "This domain cannot be registered because it contravenes the Nominet UK" && body[2] == "naming rules.  The reason is:" {
		if strings.HasSuffix(domain, ".sch.uk") && body[3] == "invalid format for a .sch.uk domain name." {
			return NotAllowed
		}
		if body[3] == "the domain name contains too many parts." {
			if delegatedUKChild(domain) {
				return NotProvided
			}
			return NotAllowed
		}
	}
	return ""
}
