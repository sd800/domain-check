package registry

import (
	"encoding/json"
	"strings"
)

// RegistrantContact keeps disclosed ownership fields together. Its structured
// components are never inferred from a free-form address or another contact.
type RegistrantContact struct {
	Name, Organization, City, StateProvince, PostalCode, Country string
	Street, Address, Phones, Emails                              []string
}

// DisplayNames keeps separately supplied name and organization fields distinct.
// An organization without a disclosed name occupies the Registrant row instead.
func (c *RegistrantContact) DisplayNames() (registrant, organization string) {
	name, organization := useful(c.Name), useful(c.Organization)
	if name == "" {
		return organization, ""
	}
	return name, organization
}

func (c *RegistrantContact) hasAddress() bool {
	return c != nil && (len(c.Street) > 0 || len(c.Address) > 0 || c.City != "" || c.StateProvince != "" || c.PostalCode != "" || c.Country != "")
}

func preserveRegistrantLayout(domain string) bool {
	if !strings.Contains(domain, ".") || strings.HasSuffix(domain, ".gov") || strings.HasSuffix(domain, ".edu") || strings.HasSuffix(domain, ".edu.cn") || strings.HasSuffix(domain, ".ac.uk") {
		return true
	}
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	return whoisProfiles[tld].has(jprsContactFields | twnicContactFields | twnicVariantNames | multilineRegistrant | ianaDomainMetadata)
}

func collectRegistrantContact(card []json.RawMessage, r *Record, named bool) {
	if preserveRegistrantLayout(r.Domain) || !named && r.RegistrantContact != nil {
		return
	}
	r.RegistrantContact = readRegistrantContact(card)
}

func readRegistrantContact(card []json.RawMessage) *RegistrantContact {
	contact := readContactCard(card, false)
	c := &RegistrantContact{Name: contact.Name, Organization: contact.Organization, Emails: contact.Emails}
	if len(card) != 2 {
		return nil
	}
	var properties [][]json.RawMessage
	if json.Unmarshal(card[1], &properties) != nil {
		return nil
	}
	for _, property := range properties {
		var kind string
		if len(property) < 4 || json.Unmarshal(property[0], &kind) != nil {
			continue
		}
		var value any
		if json.Unmarshal(property[3], &value) != nil {
			continue
		}
		switch kind {
		case "tel":
			var params struct {
				Type any `json:"type"`
			}
			_ = json.Unmarshal(property[1], &params)
			fax := false
			for _, label := range strings.Fields(strings.ToLower(vcardText(params.Type))) {
				fax = fax || label == "fax"
			}
			if !fax {
				phone := useful(vcardText(value))
				if strings.HasPrefix(strings.ToLower(phone), "tel:") {
					phone = phone[4:]
				}
				c.Phones = appendUnique(c.Phones, phone)
			}
		case "adr":
			if c.hasAddress() {
				continue
			}
			if parts, ok := value.([]any); ok && len(parts) == 7 {
				for _, part := range parts[:3] {
					c.Street = appendUnique(c.Street, vcardText(part))
				}
				c.City, c.StateProvince = useful(vcardText(parts[3])), useful(vcardText(parts[4]))
				c.PostalCode, c.Country = useful(vcardText(parts[5])), useful(vcardText(parts[6]))
				if c.Country == "" {
					c.Country = contactCountry(property[1], value)
				}
			} else if text, ok := value.(string); ok {
				c.Address = govAddressLines(property[1], text)
			}
			if !c.hasAddress() {
				c.Address = govAddressLines(property[1], "")
			}
		}
	}
	if !c.hasAddress() {
		return nil
	}
	return c
}
