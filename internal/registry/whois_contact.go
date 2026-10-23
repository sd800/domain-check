package registry

import "strings"

// Recognize field labels, not text inside values. Address/Addr remains an
// opaque multiline field; only explicitly labeled components become columns.
func whoisContactField(key string) (field string, explicit bool) {
	for _, prefix := range []string{"registrant contact ", "registrant ", "owner ", "holder "} {
		if rest, ok := strings.CutPrefix(key, prefix); ok {
			key, explicit = rest, true
			break
		}
	}
	switch key {
	case "name", "person", "personname", "full name":
		field = "name"
	case "org", "orgname", "org name", "organization", "organisation", "organization name", "organisation name", "company", "company english name", "company chinese name":
		field = "organization"
	case "street", "street address", "streetaddress", "street1", "street2", "street3", "street 1", "street 2", "street 3":
		field = "street"
	case "address", "addr", "postal address", "mailing address", "company address", "organization address", "organisation address", "address1", "address2", "address3", "address 1", "address 2", "address 3", "addr1", "addr2", "addr3", "addr 1", "addr 2", "addr 3", "locality zipcode":
		field = "address"
	case "city", "locality":
		field = "city"
	case "state", "province", "region", "state / province", "state province", "state or province":
		field = "state"
	case "postal code", "postalcode", "postcode", "zip", "zip code", "zipcode":
		field = "postal"
	case "country", "country code", "country / region":
		field = "country"
	case "phone", "phone number", "telephone", "tel":
		field = "phone"
	case "email", "e mail", "email address":
		field = "email"
	}
	return field, explicit
}

func addWHOISContact(c *RegistrantContact, field, value string) {
	value = whoisValue(value)
	switch field {
	case "name":
		if c.Name == "" {
			c.Name = value
		}
	case "organization":
		if c.Organization == "" {
			c.Organization = value
		}
	case "street":
		c.Street = appendUnique(c.Street, value)
	case "address":
		c.Address = appendUnique(c.Address, value)
	case "city":
		if c.City == "" {
			c.City = value
		}
	case "state":
		if c.StateProvince == "" {
			c.StateProvince = value
		}
	case "postal":
		if c.PostalCode == "" {
			c.PostalCode = value
		}
	case "country":
		if c.Country == "" {
			c.Country = value
		}
	case "phone":
		c.Phones = appendUnique(c.Phones, value)
	case "email":
		c.Emails = appendWHOISEmails(c.Emails, value)
	}
}

func readWHOISRegistrantContact(domain string, lines []whoisLine, owner string, embedded bool) *RegistrantContact {
	current := &RegistrantContact{}
	candidates := []*RegistrantContact{current}
	scope, pending, pendingIndent := "", "", 0
	active, nameSeen, orgSeen, plainName := true, false, false, false
	if embedded {
		scope = "registrant"
	}
	ianaOwner := strings.HasSuffix(domain, ".int")
	start := func() {
		current = &RegistrantContact{}
		candidates = append(candidates, current)
		nameSeen, orgSeen = false, false
	}
	for _, token := range lines {
		line := strings.TrimSpace(token.raw)
		if line == "" {
			pending = ""
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(line, ">>>") || line == "--" || strings.HasPrefix(lower, "terms of use") || strings.HasPrefix(lower, "copyright") || strings.HasPrefix(lower, "whois lookup made at ") {
			break
		}
		if strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") || token.object && !embedded {
			continue
		}
		key, value, labeled := token.key, token.value, token.labeled
		if whoisKeys[key] == "domain" {
			name, err := normalizeWHOISDomain(value)
			active = err == nil && name == domain
			pending, scope = "", "domain"
			if ianaOwner && active {
				scope = "registrant"
			}
			continue
		}
		if !active {
			continue
		}
		if ianaOwner && (key == "contact" || key == "remarks" || key == "source") {
			scope, pending, plainName = "other-contact", "", false
			continue
		}
		indent := len(token.raw) - len(strings.TrimLeft(token.raw, " \t"))
		if pending == "address" && indent > pendingIndent {
			_, explicit := whoisContactField(key)
			if !explicit && whoisSection(key) == "" && whoisKeys[key] == "" {
				addWHOISContact(current, pending, line)
				continue
			}
		}
		if !labeled {
			possible := whoisKey(line)
			if field, explicit := whoisContactField(possible); field != "" && (explicit || scope == "registrant") {
				key, value, labeled = possible, "", true
			} else if pending != "" && (pending != "address" && pending != "street" || indent > pendingIndent) {
				addWHOISContact(current, pending, value)
				if pending != "address" && pending != "street" {
					pending = ""
				}
				continue
			} else if plainName && scope == "registrant" {
				current.Name, plainName, nameSeen = whoisValue(value), false, true
				continue
			} else {
				pending = ""
				continue
			}
		}
		pending = ""
		if section := whoisSection(key); section != "" {
			if section == "registrant" {
				if nameSeen || orgSeen || current.hasAddress() {
					start()
				}
				scope, plainName = section, !whoisHasInlineValue(token.raw)
				if value != "" {
					current.Name, nameSeen = value, true
				}
			} else {
				scope, plainName = section, false
			}
			continue
		}
		field, explicit := whoisContactField(key)
		if field == "" || !explicit && scope != "registrant" {
			continue
		}
		plainName = false
		if field == "name" && nameSeen && value != current.Name || field == "organization" && orgSeen && value != current.Organization {
			start()
		}
		if field == "name" {
			nameSeen = true
		}
		if field == "organization" {
			orgSeen = true
		}
		addWHOISContact(current, field, value)
		if field == "address" || field == "street" || value == "" && !whoisHasInlineValue(token.raw) {
			pending, pendingIndent = field, indent
		}
	}
	for _, candidate := range candidates {
		matches := owner == "" || strings.EqualFold(owner, candidate.Name) || strings.EqualFold(owner, candidate.Organization)
		if !matches {
			continue
		}
		if candidate.hasAddress() {
			return candidate
		}
		if candidate.Name != "" || candidate.Organization != "" {
			return nil
		}
	}
	return nil
}
