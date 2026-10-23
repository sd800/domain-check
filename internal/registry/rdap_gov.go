package registry

import (
	"encoding/json"
	"strings"
)

type govAddress struct {
	street                       []string
	city, state, postal, country string
}

func (a govAddress) disclosed() bool {
	return len(a.street) > 0 || a.city != "" || a.state != "" || a.postal != "" || a.country != ""
}

// The chosen registrant supplies its own address. A later named registrant
// replaces any unnamed address, including when that named address is redacted.
func collectGovAddress(card []json.RawMessage, r *Record, named bool) {
	if !named && (len(r.RegistrantAddress) > 0 || r.RegistrantCity != "" || r.RegistrantState != "" || r.RegistrantPostalCode != "" || r.RegistrantCountry != "") {
		return
	}
	address := readGovAddress(card)
	r.RegistrantAddress = address.street
	r.RegistrantCity, r.RegistrantState = address.city, address.state
	r.RegistrantPostalCode, r.RegistrantCountry = address.postal, address.country
}

func readGovAddress(card []json.RawMessage) govAddress {
	if len(card) != 2 {
		return govAddress{}
	}
	var properties [][]json.RawMessage
	if json.Unmarshal(card[1], &properties) != nil {
		return govAddress{}
	}
	for _, property := range properties {
		var kind string
		if len(property) < 4 || json.Unmarshal(property[0], &kind) != nil || kind != "adr" {
			continue
		}
		var value any
		if json.Unmarshal(property[3], &value) != nil {
			continue
		}
		var address govAddress
		if parts, ok := value.([]any); ok && len(parts) == 7 {
			for _, part := range parts[:3] {
				address.street = appendUnique(address.street, vcardText(part))
			}
			address.city, address.state = useful(vcardText(parts[3])), useful(vcardText(parts[4]))
			address.postal, address.country = useful(vcardText(parts[5])), useful(vcardText(parts[6]))
			if address.country == "" {
				address.country = contactCountry(property[1], value)
			}
		} else if _, ok := value.(string); ok {
			address.street = govAddressLines(property[1], value.(string))
		}
		if !address.disclosed() {
			// A free-form label remains one Address block; it never supplies
			// guessed city, state, postal, or country components.
			address.street = govAddressLines(property[1], "")
		}
		if address.disclosed() {
			return address
		}
	}
	return govAddress{}
}

func govAddressLines(parameters json.RawMessage, value string) []string {
	var params struct {
		Label string `json:"label"`
	}
	_ = json.Unmarshal(parameters, &params)
	lines := func(text string) []string {
		var result []string
		for _, line := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' }) {
			result = appendUnique(result, line)
		}
		return result
	}
	if label := lines(params.Label); len(label) > 0 {
		return label
	}
	return lines(value)
}
