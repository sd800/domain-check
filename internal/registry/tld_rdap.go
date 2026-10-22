package registry

import (
	"encoding/json"
	"strings"
)

type contactCard struct {
	Name, Organization, Role, Country string
	Address, Phones, Emails           []string
}

func readContactCard(card []json.RawMessage, details bool) contactCard {
	var result contactCard
	if len(card) != 2 {
		return result
	}
	var properties [][]json.RawMessage
	if json.Unmarshal(card[1], &properties) != nil {
		return result
	}
	ambiguousCountry := false
	for _, property := range properties {
		if len(property) < 4 {
			continue
		}
		var kind string
		_ = json.Unmarshal(property[0], &kind)
		switch kind {
		case "fn", "org", "email", "adr":
		case "role", "tel":
			if !details {
				continue
			}
		default:
			continue
		}
		var value any
		_ = json.Unmarshal(property[3], &value)
		text := useful(vcardText(value))
		switch kind {
		case "fn":
			result.Name = text
		case "org":
			result.Organization = text
		case "role":
			result.Role = text
		case "email":
			result.Emails = appendUnique(result.Emails, text)
		case "tel":
			result.Phones = appendUnique(result.Phones, text)
		case "adr":
			if country := contactCountry(property[1], value); country != "" && !ambiguousCountry {
				if result.Country != "" && result.Country != country {
					result.Country, ambiguousCountry = "", true
				} else {
					result.Country = country
				}
			}
			if details && len(result.Address) == 0 {
				result.Address = contactAddress(property[1], value)
			}
		}
	}
	return result
}

func contactAddress(parameters json.RawMessage, value any) []string {
	var params struct {
		Label string `json:"label"`
	}
	_ = json.Unmarshal(parameters, &params)
	var lines []string
	add := func(text string) {
		for _, line := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' }) {
			lines = appendUnique(lines, line)
		}
	}
	if useful(params.Label) != "" {
		add(params.Label)
		return lines
	}
	if parts, ok := value.([]any); ok && len(parts) == 7 {
		for _, part := range parts[:3] {
			add(vcardText(part))
		}
		var locality []string
		for _, part := range parts[3:6] {
			if text := useful(vcardText(part)); text != "" {
				locality = append(locality, text)
			}
		}
		add(strings.Join(locality, " "))
		add(vcardText(parts[6]))
	} else {
		add(vcardText(value))
	}
	return lines
}

func collectTLDEntities(entities []rdapEntity, record *Record) {
	var registrant, admin bool
	for _, entity := range entities {
		for _, role := range entity.Roles {
			if role != "registrant" && role != "administrative" {
				continue
			}
			card := readContactCard(entity.VCardArray, true)
			if role == "registrant" && !registrant {
				name := card.Organization
				if name == "" {
					name = card.Name
				}
				if name == "" && len(card.Address) == 0 {
					continue
				}
				record.Registrant, record.RegistrantAddress = name, card.Address
				registrant = true
			} else if role == "administrative" && !admin {
				if card.Name == "" && card.Organization == "" && card.Role == "" && len(card.Phones) == 0 && len(card.Emails) == 0 {
					continue
				}
				record.Admin, record.AdminOrganization, record.AdminRole = card.Name, card.Organization, card.Role
				record.AdminPhones, record.AdminEmails = card.Phones, card.Emails
				admin = true
			}
		}
	}
}
