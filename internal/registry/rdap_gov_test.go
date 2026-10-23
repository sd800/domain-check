package registry

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const govAddressFixture = `{"objectClassName":"domain","ldhName":"example.gov","entities":[
 {"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Example Registrar"],["adr",{},"text",["","","Wrong Street","Wrong City","ZZ","12345","Wrong Country"]]]]},
 {"roles":["security"],"vcardArray":["vcard",[["fn",{},"text","REDACTED FOR PRIVACY"],["email",{},"text","security@example.test"]]]},
 {"roles":["registrant"],"vcardArray":["vcard",[["version",{},"text","4.0"],["fn",{},"text","REDACTED FOR PRIVACY"],["org",{},"text","Example Federal Office"],["adr",{},"text",["","","REDACTED FOR PRIVACY","Washington","DC","REDACTED FOR PRIVACY","United States"]],["tel",{},"text","REDACTED FOR PRIVACY"],["email",{},"text","REDACTED FOR PRIVACY"]]]}
]}`

func TestGovDisclosedRegistrantAddressAndRedaction(t *testing.T) {
	r, err := parseRDAP("example.gov", []byte(govAddressFixture))
	if err != nil || r.Registrant != "Example Federal Office" || r.RegistrantCity != "Washington" || r.RegistrantState != "DC" || r.RegistrantCountry != "United States" || len(r.RegistrantAddress) != 0 || r.RegistrantPostalCode != "" || len(r.RegistrantEmail) != 0 {
		t.Fatal(r, err)
	}
	for _, domain := range []string{"example.com", "gov"} {
		body := strings.ReplaceAll(govAddressFixture, "example.gov", domain)
		r, err := parseRDAP(domain, []byte(body))
		if err != nil || r.RegistrantCity != "" || r.RegistrantState != "" || r.RegistrantPostalCode != "" || r.RegistrantCountry != "" {
			t.Fatal("gov-specific components escaped their scope", r, err)
		}
		if domain == "gov" && (!r.TLD || !reflect.DeepEqual(r.RegistrantAddress, []string{"Washington DC", "United States"})) {
			t.Fatal("TLD address layout changed", r)
		}
		if domain == "example.com" && len(r.RegistrantAddress) != 0 {
			t.Fatal("ordinary suffix acquired gov-specific address data", r)
		}
	}
}

func govContactFixture(role, name, address string) string {
	return fmt.Sprintf(`{"roles":[%q],"vcardArray":["vcard",[["fn",{},"text",%q],["adr",{},"text",%s]]]}`, role, name, address)
}

func TestGovAddressStaysWithSelectedRegistrant(t *testing.T) {
	visible := `["","","One Street","Example City","EX","12345","Example Country"]`
	hidden := `["","","REDACTED FOR PRIVACY","","","",""]`
	for _, entities := range []string{
		govContactFixture("registrar", "Wrong Registrar", visible) + "," + govContactFixture("registrant", "Selected Owner", hidden),
		govContactFixture("technical", "Wrong Contact", visible) + "," + govContactFixture("registrant", "Selected Owner", hidden),
		govContactFixture("registrant", "Selected Owner", hidden) + "," + govContactFixture("registrant", "Another Owner", visible),
		govContactFixture("registrant", "", visible) + "," + govContactFixture("registrant", "Selected Owner", hidden),
	} {
		body := `{"objectClassName":"domain","ldhName":"example.gov","entities":[` + entities + `]}`
		r, err := parseRDAP("example.gov", []byte(body))
		if err != nil || r.Registrant != "Selected Owner" || len(r.RegistrantAddress) != 0 || r.RegistrantCity != "" || r.RegistrantState != "" || r.RegistrantPostalCode != "" || r.RegistrantCountry != "" {
			t.Fatal("address crossed contact boundaries", r, err)
		}
	}
	// If all registrant names are redacted, the first disclosed registrant
	// address remains usable without promoting an address into an owner name.
	entities := govContactFixture("registrant", "REDACTED FOR PRIVACY", visible) + "," + govContactFixture("registrant", "", `["","","Other Street","Other City","","",""]`)
	r, err := parseRDAP("example.gov", []byte(`{"objectClassName":"domain","ldhName":"example.gov","entities":[`+entities+`]}`))
	if err != nil || r.Registrant != "" || r.RegistrantCity != "Example City" || !reflect.DeepEqual(r.RegistrantAddress, []string{"One Street"}) {
		t.Fatal(r, err)
	}
}

func TestGovAddressComponentsLabelsAndMalformedValues(t *testing.T) {
	card := func(properties string) []byte {
		return []byte(`{"objectClassName":"domain","ldhName":"example.gov","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["org",{},"text","Example Office"],` + properties + `]]}]}`)
	}
	r, err := parseRDAP("example.gov", card(`["adr",{},"text",["REDACTED FOR PRIVACY","Suite 1",["One Street","REDACTED FOR PRIVACY","Building 2"],"Example City","EX","12345","United States"]]`))
	if err != nil || !reflect.DeepEqual(r.RegistrantAddress, []string{"Suite 1", "One Street Building 2"}) || r.RegistrantPostalCode != "12345" || r.RegistrantCountry != "United States" {
		t.Fatal(r, err)
	}
	r, err = parseRDAP("example.gov", card(`["adr",{"label":"REDACTED FOR PRIVACY\nPublic Address Line\nUnited States"},"text",["","","","","","",""]]`))
	if err != nil || !reflect.DeepEqual(r.RegistrantAddress, []string{"Public Address Line", "United States"}) || r.RegistrantCity != "" || r.RegistrantCountry != "" {
		t.Fatal("inferred components from free-form text", r, err)
	}
	r, err = parseRDAP("example.gov", card(`["adr",{"cc":"US"},"text",["","","","Washington","DC","","REDACTED FOR PRIVACY"]]`))
	if err != nil || r.RegistrantCountry != "US" {
		t.Fatal("lost explicitly disclosed country code", r, err)
	}
	r, err = parseRDAP("example.gov", card(`["adr",{},"text",["","","","First City","DC","",""]],["adr",{},"text",["","","Other Street","Other City","EX","12345","Other Country"]]`))
	if err != nil || r.RegistrantCity != "First City" || r.RegistrantState != "DC" || len(r.RegistrantAddress) != 0 || r.RegistrantPostalCode != "" || r.RegistrantCountry != "" {
		t.Fatal("combined separate postal addresses", r, err)
	}
	for _, value := range []string{`["wrong shape"]`, `null`, `123`, `["","","REDACTED FOR PRIVACY","REDACTED FOR PRIVACY","REDACTED FOR PRIVACY","REDACTED FOR PRIVACY","REDACTED FOR PRIVACY"]`} {
		r, err = parseRDAP("example.gov", card(`["adr",{},"text",`+value+`]`))
		if err != nil || len(r.RegistrantAddress) != 0 || r.RegistrantCity != "" || r.RegistrantCountry != "" {
			t.Fatal("malformed or redacted address became data", r, err)
		}
	}
}
