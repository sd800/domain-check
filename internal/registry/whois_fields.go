package registry

import (
	"html"
	"regexp"
	"strings"
)

// Lex each line once. Object discovery and domain/contact interpretation share
// the same labels and values; object flags replace a separate line-index map.
type whoisLine struct {
	raw, key, value        string
	labeled, object, alias bool
}

func tokenizeWHOIS(response string) []whoisLine {
	lines := make([]whoisLine, 0, min(strings.Count(response, "\n")+1, 256))
	for raw := range strings.SplitSeq(response, "\n") {
		if strings.TrimSpace(raw) == "" {
			// One blank token preserves the section boundary. Extra blank
			// lines carry no field or contact-object information.
			if len(lines) == 0 || lines[len(lines)-1].raw != "" {
				lines = append(lines, whoisLine{})
			}
			continue
		}
		key, value, labeled := splitWHOISLine(raw)
		lines = append(lines, whoisLine{raw: raw, key: key, value: value, labeled: labeled})
	}
	return lines
}

// WHOIS has no common schema. These are field labels, rather than loose
// substring matches: an administrative email must not become a registrant email.
var whoisKeys = func() map[string]string {
	groups := map[string][]string{
		"domain":       {"domain", "domain name", "domainname", "dominio", "ascii", "domain name (punycode)", "domain name(punycode)", "domain name (utf8)", "idna domain"},
		"registration": {"created", "created on", "created date", "creation date", "create date", "registration time", "registration date", "registered", "registered on", "registered date", "registered on date", "assigned", "domain record activated", "domain created", "domain create date", "domain registration date", "domain name commencement date", "first registration date", "record created on", "record created", "activation", "fecha de registro"},
		"expiration":   {"expires", "expires on", "expire", "expire date", "expiry", "expiry date", "expiration", "expiration date", "expiration time", "expiration on", "registry expiry date", "registrar registration expiration date", "paid till", "valid until", "validity", "domain expires", "domain expiration date", "record expires on", "record will expire on", "renewal date", "fecha de vencimiento"},
		"changed":      {"updated", "updated on", "updated date", "update date", "last update", "last updated", "last updated on", "last updated date", "last update time", "last modified", "last modified date", "last changed", "last changed time", "changed", "modified", "modified date", "modification date", "domain record last updated", "domain last updated date"},
		"status":       {"status", "state", "domain status", "domain state", "registration status", "query status", "registry status"},
		"registrar":    {"registrar", "registrar name", "sponsoring registrar", "current registrar", "current registar", "registration service provider"},
		"registrant":   {"registrant", "registrant name", "registrant contact name", "holder", "owner", "owner name"},
		"organization": {"registrant organization", "registrant organisation", "registrant contact organization", "registrant contact organisation", "owner orgname", "registrant name (organization)"},
		"email":        {"registrant email", "registrant e mail", "registrant contact email", "registrant contact e mail", "holder email", "owner email"},
		"nameserver":   {"name server", "name servers", "nameserver", "nameservers", "nserver", "nservers", "hostname", "host name", "dns", "dns servers", "domain servers", "domain nameservers", "domain name servers", "domain servers in listed order", "name servers information", "name server information", "name server (db)", "primary server", "secondary server"},
		"dnssec":       {"dnssec", "dnssec signed", "signed", "domain signed"},
		"ds":           {"ds", "ds record", "ds rdata", "dnskey", "signing key"},
	}
	keys := make(map[string]string)
	for field, labels := range groups {
		for _, label := range labels {
			keys[label] = field
		}
	}
	keys["exp date"] = "expiration"
	keys["domain holder organization"] = "organization"
	keys["domain holder organisation"] = "organization"
	return keys
}()

var (
	whoisDateSentence = regexp.MustCompile(`(?i)^(record created on|record expires on|record will expire on|registered on|first registration date)\s+(\d.*|before .*)$`)
	whoisDateValue    = regexp.MustCompile(`(?i)^(?:\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{4}|\d{1,2}-[a-z]{3}-\d{4})[ T]\d{1,2}:\d{2}`)
)

// These labels are deliberately recognized even when they are not output
// fields. Otherwise an empty Name can consume Address, or an omitted Email
// can consume Phone, as if the next label were the missing value.
var whoisContactLabels = map[string]bool{
	"name": true, "first name": true, "last name": true, "full name": true,
	"organization": true, "organisation": true, "organization name": true, "org": true,
	"address": true, "address1": true, "address2": true, "address3": true,
	"address 1": true, "address 2": true, "address 3": true,
	"street": true, "street address": true, "city": true, "state": true,
	"state or province": true, "province": true, "postal code": true, "postcode": true,
	"zip": true, "zip code": true, "country": true, "country code": true,
	"email": true, "e mail": true, "email address": true, "phone": true,
	"phone number": true, "telephone": true, "fax": true, "fax number": true,
	"contact id": true, "nic handle": true, "ip address": true,
}

func whoisKey(key string) string {
	key = strings.Trim(strings.TrimSpace(key), "[]* ")
	key = strings.ReplaceAll(strings.ReplaceAll(key, "_", " "), "-", " ")
	key = strings.ReplaceAll(key, "/", " / ")
	key = strings.TrimRight(key, ". ")
	key = strings.ToLower(Clean(key))
	for _, prefix := range []string{"ns", "domain server", "name server"} {
		if number, ok := strings.CutPrefix(key, prefix); ok {
			number = strings.TrimSpace(number)
			if number != "" && strings.Trim(number, "0123456789") == "" {
				return "name server"
			}
		}
	}
	if strings.HasPrefix(key, "company english name (") {
		return "company english name"
	}
	return key
}

// Read colon, dot-leader, JPRS bracket, and a small set of colonless date
// labels. Arbitrary prose is never split on whitespace into a field/value.
func splitWHOISLine(raw string) (key, value string, labeled bool) {
	line := strings.TrimSpace(raw)
	line = strings.TrimLeft(line, "* ")
	if len(line) >= 3 && (line[0] >= 'a' && line[0] <= 'z' || line[0] >= 'A' && line[0] <= 'Z') && line[1] == '.' {
		if rest := strings.TrimLeft(line[2:], " \t\n\f\r"); strings.HasPrefix(rest, "[") {
			line = rest
		}
	}
	if len(line) > 0 {
		if line[0] == 'r' || line[0] == 'R' || line[0] == 'f' || line[0] == 'F' {
			if match := whoisDateSentence.FindStringSubmatch(line); match != nil {
				return whoisKey(match[1]), useful(match[2]), true
			}
		}
		if line[0] >= '0' && line[0] <= '9' && whoisDateValue.MatchString(line) {
			return "", whoisValue(line), false
		}
	}
	if strings.HasPrefix(line, "[") {
		if end := strings.IndexByte(line, ']'); end > 0 {
			key = whoisKey(line[1:end])
			return key, whoisFieldValue(key, strings.TrimLeft(line[end+1:], ": ")), true
		}
	}
	if key, value, ok := strings.Cut(line, ":"); ok && len(key) < 180 && !strings.Contains(key, "@") && !strings.HasPrefix(key, "http") {
		// A bare NS line may contain IPv6 glue; its first colon is not
		// a label delimiter. Dot leaders on real labels remain supported.
		first := strings.Fields(key)
		if len(first) > 0 {
			if _, err := NormalizeDomain(first[0]); err == nil {
				return "", whoisValue(line), false
			}
		}
		key = whoisKey(key)
		return key, whoisFieldValue(key, value), true
	}
	key = whoisKey(line)
	if whoisSection(key) != "" || whoisKeys[key] != "" || whoisContactLabels[key] {
		return key, "", true
	}
	return "", whoisValue(line), false
}

func whoisHasInlineValue(raw string) bool {
	line := strings.TrimSpace(raw)
	if end := strings.IndexByte(line, ']'); strings.Contains(line, "[") && end >= 0 {
		return strings.Trim(strings.TrimSpace(line[end+1:]), ": ") != ""
	}
	_, value, ok := strings.Cut(line, ":")
	return ok && strings.TrimSpace(value) != ""
}

func whoisValue(value string) string {
	// Some older registries emit HTML entities or fill the value with dots.
	return useful(html.UnescapeString(strings.TrimLeft(strings.TrimSpace(value), ".")))
}

func whoisFieldValue(key, value string) string {
	if whoisKeys[key] == "domain" {
		// Identity validation must see the original label characters, including
		// emoji ZWJ. Sanitizing them first can turn it into a different name.
		return html.UnescapeString(strings.TrimLeft(strings.TrimSpace(value), "."))
	}
	if whoisKeys[key] == "status" && strings.EqualFold(strings.TrimSpace(value), "not available") {
		return Clean(value) // This is a real registered-domain status in .be.
	}
	return whoisValue(value)
}

// A section is kept across blank lines; several registries put a blank line
// between a heading and its first value. Only explicit headings change scope.
func whoisSection(key string) string {
	switch key {
	case "registrant", "holder", "owner", "titular", "registrant contact", "registrant contact information", "owner contact", "domain owners", "domain owner", "domain registrant", "domain owners / registrant", "domain owner / registrant":
		return "registrant"
	case "domain managers":
		return "manager"
	case "registrar":
		return "registrar"
	case "administrative contact information", "admin contact information", "technical contact information", "tech contact information", "billing contact information", "administrative contact", "administrative contacts", "administrativ contact", "administrative", "admin contact", "admin c", "technical contact", "technical contacts", "technical", "tech contact", "tech", "tech c", "zone c", "billing contact", "billing", "contacto administrativo", "contacto tecnico", "contacto financiero", "registrar technical contacts", "registrar contact information", "reseller":
		return "other-contact"
	case "name server", "name servers", "nameservers", "nservers", "dns servers", "domain servers", "domain nameservers", "domain name servers", "domain servers in listed order", "name servers information", "name server information":
		return "nameserver"
	case "domain", "domain information", "domain name information", "domain details", "relevant dates", "additional info", "details":
		return "domain"
	case "whois", "whois server", "disclaimer", "status information", "flags", "contact information":
		return "metadata"
	case "keys", "dnssec ds records":
		return "keys"
	}
	return ""
}
