package registry

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestRegistrarCountryUsesOnlyItsDisclosedAddress(t *testing.T) {
	for _, tc := range []struct {
		name, params, country, want string
	}{
		{"cc", `{"cc":"US"}`, "", "Example Registrar [US]"},
		{"lowercase", `{"cc":"gb"}`, "", "Example Registrar [GB]"},
		{"legacy country component", `{}`, "JP", "Example Registrar [JP]"},
		{"cc wins", `{"cc":"US"}`, "GB", "Example Registrar [US]"},
		{"country name is not a code", `{}`, "United States", "Example Registrar"},
		{"label is not a code", `{"label":"US"}`, "", "Example Registrar"},
		{"malformed", `{"cc":"USA"}`, "", "Example Registrar"},
		{"bad type", `{"cc":["US"]}`, "", "Example Registrar"},
		{"control characters", `{"cc":"U\u001bS"}`, "", "Example Registrar"},
		{"contradictory parameters", `{"cc":"US","CC":"GB"}`, "", "Example Registrar"},
		{"legacy case", `{"CC":"CA"}`, "", "Example Registrar [CA]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address := fmt.Sprintf(`["adr",%s,"text",["","","","","","",%q]]`, tc.params, tc.country)
			body := `{"objectClassName":"domain","ldhName":"example.test","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Example Owner"],["adr",{"cc":"DE"},"text",["","","","","","",""]]]]},{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Example Registrar"],` + address + `]]}]}`
			r, err := parseRDAP("example.test", []byte(body))
			if err != nil || r.Registrar != tc.want || r.Registrant != "Example Owner" {
				t.Fatal(r, err)
			}
		})
	}
}

func TestRegistrarCountryCannotBeTakenFromAnotherEntity(t *testing.T) {
	for _, body := range []string{
		`{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Example Registrar"]]],"entities":[{"roles":["abuse"],"vcardArray":["vcard",[["adr",{"cc":"US"},"text",["","","","","","",""]]]]}]}`,
		`{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Example Registrar"]]]},{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Other Registrar"],["adr",{"cc":"US"},"text",["","","","","","",""]]]]}`,
	} {
		r, err := parseRDAP("example.test", []byte(`{"objectClassName":"domain","ldhName":"example.test","entities":[`+body+`]}`))
		if err != nil || r.Registrar != "Example Registrar" {
			t.Fatal(r, err)
		}
	}
	for _, name := range []string{"", "REDACTED FOR PRIVACY"} {
		body := fmt.Sprintf(`{"objectClassName":"domain","ldhName":"example.test","entities":[{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text",%q],["adr",{"cc":"US"},"text",["","","","","","",""]]]]}]}`, name)
		if r, err := parseRDAP("example.test", []byte(body)); err != nil || r.Registrar != "" {
			t.Fatal("country became a registrar name", r, err)
		}
	}
}

func TestConflictingRegistrarCountriesAndTLDCountryAreNotAnnotated(t *testing.T) {
	entity := `{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Example Registrar"],["adr",{"cc":"US"},"text",["","","","","","",""]],["adr",{"cc":"GB"},"text",["","","","","","",""]]]]}`
	r, err := parseRDAP("example.test", []byte(`{"objectClassName":"domain","ldhName":"example.test","entities":[`+entity+`]}`))
	if err != nil || r.Registrar != "Example Registrar" {
		t.Fatal(r, err)
	}
	body := strings.ReplaceAll(entity, `"registrar"`, `"registrant"`)
	r, err = parseRDAP("zz", []byte(`{"objectClassName":"domain","ldhName":"zz","entities":[`+body+`]}`))
	if err != nil || !r.TLD || r.Registrant != "Example Registrar" || r.Registrar != "" || strings.Contains(r.Registrant, "[") {
		t.Fatal("TLD acquired a country annotation", r, err)
	}
	// Invalid parameter JSON never supplies a country through its value instead.
	if contactCountry(json.RawMessage(`[]`), []any{"", "", "", "", "", "", "US"}) != "" {
		t.Fatal("malformed address accepted")
	}
}
