package registry

import (
	"regexp"
	"strings"
)

var reservedResponse = regexp.MustCompile(`(?i)^(?:this )?(?:domain(?: name)?(?:\s+\S+)?\s+(?:is )?reserved\b|reserved(?: domain| name)?[.!]?$)`)
var forbiddenResponse = regexp.MustCompile(`(?i)^(?:this )?(?:domain(?: name)?(?:\s+\S+)?\s+(?:is )?(?:not allowed|not permitted|prohibited)(?: for registration)?[.!]?$|registration (?:is )?(?:not allowed|not permitted|prohibited)[.!]?$)`)

func negativeWHOIS(line string) Result {
	line = strings.ToLower(strings.TrimSpace(line))
	if reservedResponse.MatchString(line) {
		return Reserved
	}
	if forbiddenResponse.MatchString(line) {
		return NotAllowed
	}
	for _, prefix := range []string{"no match for ", "no matching record", "no entries found", "no data found", "no information available about domain name", "not found:", "domain not found", "no such domain", "no object found", "the queried object does not exist:"} {
		if strings.HasPrefix(line, prefix) {
			return NoRecord
		}
	}
	if line == "not found" || line == "not found." || line == "no entries found." {
		return NoRecord
	}
	if line == "status: free" || line == "status: available" || line == "domain is not registered" || line == "domain not registered" {
		return NotRegistered
	}
	return ""
}

type whoisObject struct {
	kind, handle, name, organization string
	emails, nameservers              []string
	signed                           bool
}

// Resolve only objects already included in the registry response. No handle,
// referral, contact URL, or nameserver here ever initiates another query.
func whoisObjects(lines []string) (map[string]whoisObject, map[int]bool) {
	objects := make(map[string]whoisObject)
	skip := make(map[int]bool)
	for start := 0; start < len(lines); {
		end := start
		for end < len(lines) && strings.TrimSpace(lines[end]) != "" {
			end++
		}
		var object whoisObject
		domainBlock := false
		for i := start; i < end; i++ {
			key, value, ok := splitWHOISLine(lines[i])
			if !ok || value == "" {
				continue
			}
			if whoisKeys[key] == "domain" {
				domainBlock = true
			}
			switch key {
			case "nic hdl", "contact", "domain registrant":
				object.kind, object.handle = "contact", value
			case "nsset", "keyset":
				if object.handle == "" {
					object.kind, object.handle = key, value
				}
			case "registrar":
				if i == start {
					object.kind, object.handle = "registrar", value
				}
			}
		}
		if !domainBlock && object.handle != "" {
			for i := start; i < end; i++ {
				key, value, ok := splitWHOISLine(lines[i])
				if !ok || value == "" {
					continue
				}
				switch key {
				case "name", "person", "personname":
					if object.name == "" {
						object.name = value
					}
				case "org", "organization", "organisation", "organization name":
					if object.organization == "" {
						object.organization = value
					}
				case "email", "e mail":
					object.emails = appendWHOISEmails(object.emails, value)
				}
				if whoisKeys[key] == "nameserver" {
					object.nameservers = appendWHOISNameservers(object.nameservers, value)
				}
				if whoisKeys[key] == "ds" && validWHOISKey(value) {
					object.signed = true
				}
			}
			// A registrar label alone is a domain field, not a contact object.
			if object.kind != "registrar" || object.name != "" || object.organization != "" {
				objects[whoisObjectID(object.kind, object.handle)] = object
				for i := start; i < end; i++ {
					skip[i] = true
				}
			}
		}
		start = end + 1
	}
	return objects, skip
}

func parseWHOIS(domain, response string) (Record, error) {
	r := Record{Domain: domain}
	lines := strings.Split(response, "\n")
	objects, skip := whoisObjects(lines)
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	var scope, pending, organization, holder, nsset, keyset string
	var found, mismatched, ignoredDomain, plainName bool
	contactNameSeen := false
	blank, scopeIndent := false, 0
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			blank = true
			continue
		}
		if found && (strings.HasPrefix(line, ">>>") || strings.HasPrefix(strings.ToLower(line), "terms of use:") || strings.HasPrefix(strings.ToLower(line), "copyright notice:")) {
			break // Footer prose is not another availability determination.
		}
		if strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">>>") || strings.HasPrefix(line, "***") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "limit exceeded") || strings.Contains(lower, "quota exceeded") || strings.HasPrefix(lower, "too many queries") || strings.HasPrefix(lower, "access denied") {
			return Record{}, queryError("whois-failed", line)
		}
		if result := negativeWHOIS(line); result != "" {
			if found && result == NoRecord && (scope == "registrant" || scope == "other-contact" || scope == "registrar") {
				continue // An included contact object can be missing independently.
			}
			r.Result = result
			continue
		}
		key, value, labeled := splitWHOISLine(raw)
		if scope == "nameserver" && strings.Trim(line, "-= \t") == "" {
			continue // A few registries put a table rule before the first NS.
		}
		if skip[i] {
			if key == "domain registrant" {
				holder = value
			}
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		field := whoisKeys[key]
		if labeled && blank && (field == "registration" || field == "expiration" || field == "changed" || field == "dnssec") && (scope == "nameserver" && tld != "ee" || scope == "registrar" && indent <= scopeIndent) {
			scope = "domain"
		}
		blank = false
		if !labeled {
			if pending != "" {
				if pending == "nameserver" && !whoisNameserverLine(value) {
					pending, scope = "", "metadata"
					continue
				}
				key = pending
				if pending != "nameserver" && pending != "status" {
					pending = ""
				}
			} else if scope == "nameserver" {
				if !whoisNameserverLine(value) {
					scope = "metadata"
					continue
				}
				key = "nameserver"
			} else if scope == "registrant" && plainName {
				// Only the first unlabelled line of an explicit owner section
				// is a name. Addresses and phone numbers are not names.
				key, plainName = "registrant name", false
			} else if scope == "registrar" && plainName {
				key, plainName = "registrar name", false
			} else {
				continue
			}
		} else {
			pending = ""
			if scope == "registrant" || scope == "registrar" {
				plainName = false // Once labels begin, never fall back to prose.
			}
		}
		field = whoisKeys[key]
		if key == "domain" && value == "" {
			scope, pending = "domain", "domain"
			continue
		}
		if field == "domain" && value != "" {
			normalized, err := normalizeWHOISDomain(value)
			if err != nil || normalized != domain {
				ignoredDomain, mismatched = true, true
				continue
			}
			ignoredDomain, found, scope = false, true, "domain"
			continue
		}
		if ignoredDomain {
			continue
		}
		section := whoisSection(key)
		if (key == "admin c" || key == "tech c" || key == "zone c") && value != "" {
			continue // A handle reference is not a contact-section heading.
		}
		if tld == "jp" && (key == "administrative contact" || key == "technical contact") {
			continue // JPRS bracket fields contain handles, including empty ones.
		}
		if section == "other-contact" && strings.HasPrefix(value, "http") {
			continue // A contact URL is a reference, not an embedded object.
		}
		if key == "contact information" && tld == "jp" {
			section = "registrant"
		}
		if section != "" && (value == "" || section == "other-contact" || key == "domain registrant") {
			scope, plainName = section, (section == "registrant" || section == "registrar") && !whoisHasInlineValue(raw)
			contactNameSeen = false
			scopeIndent = indent
			if field == "domain" || field == "nameserver" || field == "status" {
				pending = field
			}
			continue
		}
		if field == "registrant" && key == "registrant" || field == "nameserver" && labeled && value != "" && scope == "other-contact" {
			scope = "domain"
		}
		if key == "contact" && (lower == "contact: administrative" || lower == "contact: technical") {
			scope = "other-contact"
			continue
		}
		if key == "holder c" || key == "owner c" {
			holder = value
			continue
		}
		if key == "nsset" || key == "keyset" {
			if key == "nsset" {
				nsset = value
			} else {
				keyset = value
			}
			continue
		}
		// Domain-specific labels resume domain scope after contact sections.
		if strings.HasPrefix(key, "domain ") && (field == "registration" || field == "expiration" || field == "changed" || field == "status") || strings.HasPrefix(key, "record ") || strings.HasPrefix(key, "fecha de ") {
			scope = "domain"
		}
		if field == "expiration" && indent <= scopeIndent && (key == "expiry date" || key == "registry expiry date" || key == "registrar registration expiration date") {
			scope = "domain"
		}
		if key == "admin e mail" || key == "tech e mail" {
			scope = "other-contact"
		}
		if key == "original created" {
			r.Registration = formatWHOISTime(tld, value)
			continue
		}
		if tld == "am" && (key == "registered" || key == "last modified") {
			scope = "domain"
		}
		if key == "name" || key == "nombre" || key == "person" || key == "personname" {
			if labeled && scope == "registrant" {
				if contactNameSeen {
					scope = "other-contact"
				}
				contactNameSeen = true
			}
			switch scope {
			case "domain":
				if key == "name" {
					normalized, err := NormalizeDomain(value)
					if err != nil || normalized != domain {
						mismatched, ignoredDomain = true, true
						continue
					}
					found = true
				}
			case "registrant":
				field = "registrant"
			case "registrar":
				field = "registrar"
			case "nameserver":
				field = "nameserver"
			default:
				if key == "person" || key == "personname" {
					field = "registrant"
				}
			}
		}
		if key == "org" || key == "organization" || key == "organisation" || key == "organization name" || key == "company english name" || key == "company chinese name" {
			if scope == "registrar" {
				field = "registrar"
			} else if scope == "registrant" || scope == "domain" || scope == "" {
				field = "organization"
			}
		}
		if key == "email" || key == "e mail" || key == "email address" {
			if scope == "registrant" || (scope == "domain" || scope == "") && (tld == "by" || tld == "il") {
				field = "email"
			}
		}
		if key == "dnssec" && tld == "tn" {
			scope = "domain"
		}
		if value == "" {
			if field != "" && !whoisHasInlineValue(raw) {
				pending = key
			}
			continue
		}
		// Explicit domain fields (Name Server, Registrant Email...) may occur
		// after a contact block. Generic object dates and DNSSEC capabilities
		// must remain scoped to that object.
		contactScope := scope == "registrant" || scope == "registrar" || scope == "other-contact" || scope == "nameserver" || scope == "metadata"
		switch field {
		case "registration", "expiration", "changed":
			if contactScope {
				continue
			}
			date := formatWHOISTime(tld, value)
			switch field {
			case "registration":
				if r.Registration == "" {
					r.Registration = date
				}
			case "expiration":
				if r.Expiration == "" || key == "registry expiry date" {
					r.Expiration = date
				}
			case "changed":
				if r.Changed == "" || tld == "il" && date > r.Changed {
					r.Changed = date
				}
			}
		case "status":
			if scope != "domain" && scope != "" && key != "domain status" {
				continue
			}
			if index := strings.Index(value, " http"); index >= 0 {
				value = value[:index]
			}
			if strings.EqualFold(value, "not available") {
				if !strings.Contains(strings.ToLower(strings.Join(r.Statuses, "\n")), "not available") {
					r.Statuses = append(r.Statuses, value)
				}
			} else {
				r.Statuses = appendUnique(r.Statuses, value)
			}
			pending = ""
		case "registrar":
			if (scope != "other-contact" || key == "registrar name") && r.Registrar == "" {
				r.Registrar = value
			}
		case "registrant":
			if scope == "other-contact" || scope == "registrar" {
				continue
			}
			if _, ok := objects[whoisObjectID("contact", value)]; ok || key == "holder" && (tld == "se" || tld == "nu") || key == "registrant" && strings.Contains(" at ir mw mk ls ve ", " "+tld+" ") {
				holder = value
			} else if r.Registrant == "" {
				r.Registrant = value
			}
		case "organization":
			if scope != "other-contact" && scope != "registrar" && organization == "" {
				organization = value
			}
		case "email":
			if scope != "other-contact" && scope != "registrar" {
				r.RegistrantEmail = appendWHOISEmails(r.RegistrantEmail, value)
			}
		case "nameserver":
			r.Nameservers = appendWHOISNameservers(r.Nameservers, value)
			if !labeled {
				pending = "nameserver"
			}
		case "dnssec":
			if !contactScope && r.DNSSEC == "" {
				r.DNSSEC = normalizeWHOISDNSSEC(value)
			}
		case "ds":
			candidate := value
			// JPRS wraps the digest in parentheses onto following lines.
			if strings.HasSuffix(candidate, "(") {
				for j := i + 1; j < len(lines); j++ {
					if _, _, label := splitWHOISLine(lines[j]); label {
						break
					}
					candidate += " " + strings.Trim(strings.TrimSpace(lines[j]), "() ")
					if strings.Contains(lines[j], ")") {
						break
					}
				}
				candidate = strings.ReplaceAll(candidate, "(", " ")
			}
			if (scope == "domain" || scope == "keys" || scope == "") && validWHOISKey(candidate) {
				r.DNSSEC = "signedDelegation"
			}
		}
		if scope == "keys" && (key == "flags" || key == "keytag") && whoisPublicKey.MatchString(value) {
			r.DNSSEC = "signedDelegation"
		}
	}
	if r.Result != "" {
		if found && (r.Registration != "" || len(r.Nameservers) > 0) {
			return Record{}, queryError("invalid-response", "contradictory WHOIS response")
		}
		return Record{Domain: domain, Result: r.Result}, nil
	}
	// NIC Chile deliberately omits a domain-name line in positive responses.
	// Require its complete positive-record signature, and no conflicting name.
	if !found && !mismatched && tld == "cl" && r.Registrant != "" && r.Registrar != "" && r.Registration != "" && r.Expiration != "" && len(r.Nameservers) > 0 {
		found = true
	}
	if !found {
		return Record{}, queryError("invalid-response", "unrecognized or mismatched WHOIS response")
	}
	if object, ok := objects[whoisObjectID("contact", holder)]; ok {
		r.Registrant, organization = object.name, object.organization
		r.RegistrantEmail = object.emails
	}
	if organization != "" {
		r.Registrant = organization
	}
	var registrarObject whoisObject
	registrarCount := 0
	for _, object := range objects {
		if object.kind == "registrar" {
			registrarCount++
			if r.Registrar == "" || strings.EqualFold(r.Registrar, object.handle) {
				registrarObject = object
			}
		}
	}
	if registrarCount == 1 && registrarObject.handle != "" {
		r.Registrar = registrarObject.organization
		if r.Registrar == "" {
			r.Registrar = registrarObject.name
		}
	}
	if object, ok := objects[whoisObjectID("nsset", nsset)]; ok {
		for _, ns := range object.nameservers {
			r.Nameservers = appendUnique(r.Nameservers, ns)
		}
	}
	if object, ok := objects[whoisObjectID("keyset", keyset)]; ok && object.signed {
		r.DNSSEC = "signedDelegation"
	}
	return r, nil
}

func whoisObjectID(kind, handle string) string {
	return kind + ":" + strings.ToLower(handle)
}

func normalizeWHOISDomain(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) > 1 {
		rest := strings.Join(parts[1:], " ")
		// Bulgaria appends a parenthesized alternate spelling. Hong Kong
		// lists bundled names on the same line as the primary domain.
		if strings.HasPrefix(rest, "(") && strings.HasSuffix(rest, ")") || strings.HasPrefix(strings.ToLower(rest), "bundled domain name:") {
			value = parts[0]
		}
	}
	return NormalizeDomain(value)
}

func appendWHOISNameservers(nameservers []string, value string) []string {
	// Glue addresses, parentheses, commas, and ordinal prefixes are common.
	for _, token := range strings.FieldsFunc(value, func(r rune) bool { return strings.ContainsRune(" ,;()\t", r) }) {
		if strings.ContainsAny(token, ":/@") {
			continue
		}
		if ns, err := NormalizeDomain(token); err == nil {
			nameservers = appendUnique(nameservers, ns)
		}
	}
	return nameservers
}

func whoisNameserverLine(value string) bool {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return false
	}
	_, err := NormalizeDomain(strings.Trim(parts[0], ",;()"))
	return err == nil
}

var whoisEmail = regexp.MustCompile(`(?i)[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?\.[a-z]{2,}`)
var whoisPublicKey = regexp.MustCompile(`(?i)\bprotocol:\s*3\s+algorithm:\s*[a-z0-9_-]+\s+pubkey:\s*[a-z0-9+/=]{8,}`)

func appendWHOISEmails(emails []string, value string) []string {
	for _, email := range whoisEmail.FindAllString(value, -1) {
		emails = appendUnique(emails, email)
	}
	return emails
}

func normalizeWHOISDNSSEC(value string) string {
	switch strings.ToLower(value) {
	case "yes", "true", "signed", "active", "signed delegation", "signeddelegation":
		return "signedDelegation"
	case "no", "false", "unsigned", "inactive", "not signed", "unsigned delegation":
		return "unsigned"
	}
	return value
}

func validWHOISKey(value string) bool {
	// A DS/DNSKEY record has numeric metadata followed by a key or digest.
	parts := strings.Fields(value)
	if len(parts) < 4 {
		return false
	}
	for _, part := range parts[:3] {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return false
		}
	}
	return len(parts[3]) >= 8
}
