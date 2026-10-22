package registry

import (
	"regexp"
	"strings"
)

type whoisObject struct {
	kind, handle, name, organization string
	emails, nameservers              []string
	signed                           bool
}

// Resolve only objects already included in the registry response. No handle,
// referral, contact URL, or nameserver here ever initiates another query.
func whoisObjects(lines []whoisLine) map[string]whoisObject {
	var objects map[string]whoisObject
	for start := 0; start < len(lines); {
		end := start
		for end < len(lines) && strings.TrimSpace(lines[end].raw) != "" {
			end++
		}
		var object whoisObject
		domainBlock := false
		for i := start; i < end; i++ {
			key, value, ok := lines[i].key, lines[i].value, lines[i].labeled
			if !ok || value == "" {
				continue
			}
			if whoisKeys[key] == "domain" || lines[i].alias {
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
				key, value, ok := lines[i].key, lines[i].value, lines[i].labeled
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
				if objects == nil {
					objects = make(map[string]whoisObject)
				}
				objects[whoisObjectID(object.kind, object.handle)] = object
				for i := start; i < end; i++ {
					lines[i].object = true
				}
			}
		}
		start = end + 1
	}
	return objects
}

func parseWHOIS(domain, response string) (Record, error) {
	r := Record{Domain: domain}
	if strings.EqualFold(strings.TrimRight(strings.TrimSpace(response), ".!"), "reserved") {
		return Record{Domain: domain, Result: Reserved}, nil
	}
	lines := tokenizeWHOIS(response)
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	profile := whoisProfiles[tld]
	if err := markWHOISBundle(domain, lines, profile); err != nil {
		return Record{}, err
	}
	objects := whoisObjects(lines)
	var scope, pending, organization, holder, nsset, keyset string
	var found, mismatched, ignoredDomain, plainName bool
	contactNameSeen := false
	var manager whoisObject
	explicitRegistrant := false
	blank, scopeIndent := false, 0
	for i, token := range lines {
		raw := token.raw
		line := strings.TrimSpace(raw)
		if line == "" {
			blank = true
			continue
		}
		if (!found || strings.HasPrefix(line, ">>>")) && whoisFailure(line) {
			return Record{}, queryError("whois-failed", line)
		}
		if strings.HasPrefix(line, ">>>") && !mismatched {
			message := strings.TrimSpace(strings.TrimPrefix(line, ">>>"))
			switch strings.ToLower(message) {
			case "the domain contains special characters not allowed.", "this domain violates registry policy.":
				r.Result = NotAllowed
				continue
			}
			if result := negativeWHOIS(message); result != "" {
				r.Result = result
				continue
			}
		}
		if (found || r.Result != "") && (strings.HasPrefix(line, ">>>") || strings.HasPrefix(strings.ToLower(line), "terms of use:") || strings.HasPrefix(strings.ToLower(line), "copyright notice:")) {
			break // Footer prose is not another availability determination.
		}
		if strings.HasPrefix(line, "%") || strings.HasPrefix(line, ">>") || strings.HasPrefix(line, "***") {
			// Some registries use protocol comments for the entire negative
			// reply. Recognize only explicit error forms before any record.
			if !found {
				message := strings.TrimLeft(line, "%>* ")
				if match := commentedWHOISError.FindStringSubmatch(message); match != nil {
					message = match[1]
				}
				if result := negativeWHOISFor(domain, message); result != "" {
					r.Result = result
				}
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "limit exceeded") || strings.Contains(lower, "quota exceeded") || strings.HasPrefix(lower, "too many queries") || strings.HasPrefix(lower, "access denied") || strings.HasPrefix(lower, "requests of this client are not permitted") || lower == "this tld has no whois server." {
			return Record{}, queryError("whois-failed", line)
		}
		if result := negativeWHOISFor(domain, line); result != "" {
			if found && result == NoRecord && (scope == "registrant" || scope == "other-contact" || scope == "registrar" || scope == "manager") {
				continue // An included contact object can be missing independently.
			}
			r.Result = result
			continue
		}
		key, value, labeled := token.key, token.value, token.labeled
		if profile.has(lithuanianContacts) {
			switch key {
			case "contact organization":
				key, scope = "registrant organization", "registrant"
			case "contact email":
				key, scope = "registrant email", "registrant"
			}
		}
		if profile.has(luxembourgStatus) && key == "domaintype" {
			key = "domain status"
		}
		if profile.has(ugandanDomainUpdate) && key == "information last updated" {
			key, scope = "domain last updated date", "domain"
		}
		if token.alias {
			ignoredDomain, found, scope = false, true, "domain"
			continue
		}
		if scope == "nameserver" && strings.Trim(line, "-= \t") == "" {
			continue // A few registries put a table rule before the first NS.
		}
		if token.object {
			if key == "domain registrant" {
				holder = value
			}
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		field := whoisKeys[key]
		if profile.has(lithuanianContacts) && field == "nameserver" {
			scope = "domain" // The contact block ends before domain NS and DS fields.
		}
		if labeled && blank && (field == "registration" || field == "expiration" || field == "changed" || field == "dnssec") && (scope == "nameserver" && !profile.has(nameserverObjectDates) || scope == "registrar" && indent <= scopeIndent) {
			scope = "domain"
		}
		blank = false
		if !labeled {
			if pending != "" {
				if pending == "domain" {
					value = whoisFieldValue("domain", raw)
				}
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
			} else if profile.has(twnicVariantNames) && scope == "registrant" && value != "" && whoisEmail.FindString(value) == value {
				key, plainName = "registrant email", false
			} else if (scope == "registrant" || scope == "manager") && plainName {
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
			if scope == "registrant" || scope == "registrar" || scope == "manager" {
				plainName = false // Once labels begin, never fall back to prose.
			}
		}
		field = whoisKeys[key]
		if field == "domain" && value == "" {
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
		if profile.has(jprsContactFields) && (key == "administrative contact" || key == "technical contact") {
			continue // JPRS bracket fields contain handles, including empty ones.
		}
		if section == "other-contact" && strings.HasPrefix(value, "http") {
			continue // A contact URL is a reference, not an embedded object.
		}
		if key == "contact information" && profile.has(jprsContactFields) {
			section = "registrant"
		}
		if key == "whois server" && strings.Contains(raw, ":") && !strings.HasPrefix(strings.TrimSpace(raw), "**") {
			continue // A flat referral field, even empty, is not a metadata section.
		}
		if section != "" && (value == "" || section == "other-contact" || key == "domain registrant") {
			scope, plainName = section, (section == "registrant" || section == "registrar" || section == "manager") && !whoisHasInlineValue(raw)
			if section == "registrant" {
				explicitRegistrant = true
			}
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
		if profile.has(ianaDomainMetadata) && (key == "remarks" || key == "source") {
			scope = "domain"
		}
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
		if profile.has(domainDatesAfterContact) && (key == "registered" || key == "last modified") {
			scope = "domain"
		}
		if key == "name" || key == "nombre" || key == "person" || key == "personname" {
			if labeled && (scope == "registrant" || scope == "manager") {
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
			case "registrant", "manager":
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
			} else if scope == "registrant" || scope == "manager" || scope == "domain" || scope == "" {
				field = "organization"
			}
		}
		if key == "email" || key == "e mail" || key == "email address" {
			if scope == "registrant" || scope == "manager" || (scope == "domain" || scope == "") && profile.has(domainContactEmail) {
				field = "email"
			}
		}
		if key == "dnssec" && profile.has(domainDNSSECAfterContact) {
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
		contactScope := scope == "registrant" || scope == "registrar" || scope == "other-contact" || scope == "nameserver" || scope == "metadata" || scope == "manager"
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
				if r.Changed == "" || profile.has(changeHistoryDates) && date > r.Changed {
					r.Changed = date
				}
			}
		case "status":
			if scope != "domain" && scope != "" && key != "domain status" {
				continue
			}
			if profile.has(bulgarianStatus) && strings.EqualFold(value, "invalid") {
				r.Result = NotAllowed
				continue
			}
			if result := negativeWHOISStatus(value); result != "" {
				r.Result = result
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
			if scope == "manager" {
				if manager.name == "" {
					manager.name = value
				}
				continue
			}
			if scope == "other-contact" || scope == "registrar" {
				continue
			}
			explicitRegistrant = true
			if _, ok := objects[whoisObjectID("contact", value)]; ok || key == "holder" && profile.has(holderHandle) || key == "registrant" && profile.has(registrantHandle) {
				holder = value
			} else if r.Registrant == "" {
				r.Registrant = value
			}
		case "organization":
			if scope == "manager" {
				if manager.organization == "" {
					manager.organization = value
				}
				continue
			}
			if scope != "other-contact" && scope != "registrar" && organization == "" {
				explicitRegistrant = true
				organization = value
			}
		case "email":
			if scope == "manager" {
				manager.emails = appendWHOISEmails(manager.emails, value)
				continue
			}
			if scope != "other-contact" && scope != "registrar" {
				explicitRegistrant = true
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
				if profile.has(hkBundledNames) {
					r.DNSSEC = normalizeBundledDNSSEC(value)
				}
			}
		case "ds":
			candidate := value
			// JPRS wraps the digest in parentheses onto following lines.
			if strings.HasSuffix(candidate, "(") {
				for j := i + 1; j < len(lines); j++ {
					if lines[j].labeled {
						break
					}
					candidate += " " + strings.Trim(strings.TrimSpace(lines[j].raw), "() ")
					if strings.Contains(lines[j].raw, ")") {
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
	if found && ignoredDomain {
		return Record{}, queryError("invalid-response", "contradictory WHOIS domain identities")
	}
	if r.Result != "" {
		if mismatched && !found {
			return Record{}, queryError("invalid-response", "response domain does not match the query")
		}
		if found && (r.Registration != "" || len(r.Nameservers) > 0) {
			return Record{}, queryError("invalid-response", "contradictory WHOIS response")
		}
		return Record{Domain: domain, Result: r.Result}, nil
	}
	// NIC Chile deliberately omits a domain-name line in positive responses.
	// Require its complete positive-record signature, and no conflicting name.
	if !found && !mismatched && profile.has(implicitDomainIdentity) && r.Registrant != "" && r.Registrar != "" && r.Registration != "" && r.Expiration != "" {
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
	// Legacy manager-only replies remain useful. A separate owner section,
	// even a redacted one, always takes precedence over management contacts.
	if !explicitRegistrant && r.Registrant == "" && (manager.name != "" || manager.organization != "" || len(manager.emails) > 0) {
		r.Registrant = manager.organization
		if r.Registrant == "" {
			r.Registrant = manager.name
		}
		r.RegistrantEmail = manager.emails
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
