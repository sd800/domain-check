package registry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const detailedWHOISFixture = `Domain Name: example.us
Registrant Name: Alex Example
Registrant Organization: Example City Authority
Registrant Street: 400 Example Street
Registrant Street:
Registrant Street:
Registrant City: Example City
Registrant State/Province: IL
Registrant Postal Code: 60540
Registrant Country: US
Registrant Phone: +1.5550100
Registrant Phone Ext: 123
Registrant Fax: +1.5550199
Registrant Email: owner@example.test
Admin Name: Wrong Admin
Admin Street: Wrong Street
Admin Phone: +1.5550999
Admin Email: wrong@example.test
Name Server: ns1.example.test
`

func TestWHOISRegistrantComponentsAndContactIsolation(t *testing.T) {
	r, err := parseWHOIS("example.us", detailedWHOISFixture)
	want := &RegistrantContact{Name: "Alex Example", Organization: "Example City Authority", Street: []string{"400 Example Street"}, City: "Example City", StateProvince: "IL", PostalCode: "60540", Country: "US", Phones: []string{"+1.5550100"}, Emails: []string{"owner@example.test"}}
	if err != nil || !reflect.DeepEqual(r.RegistrantContact, want) {
		t.Fatalf("contact: %+v, error: %v", r.RegistrantContact, err)
	}
	// The stable legacy fields remain available; detailed presentation uses
	// the explicit Name and Organization separately.
	if r.Registrant != "Example City Authority" || len(r.Nameservers) != 1 {
		t.Fatal(r)
	}
	for _, domain := range []string{"example.gov", "example.edu", "example.com.tw", "example.ac.uk", "example.edu.cn"} {
		body := strings.ReplaceAll(detailedWHOISFixture, "example.us", domain)
		r, err := parseWHOIS(domain, body)
		if err != nil || r.RegistrantContact != nil {
			t.Fatal("replaced a dedicated layout", domain, r, err)
		}
	}
}

func TestWHOISWholeAddressRemainsMultilineAndOpaque(t *testing.T) {
	for _, label := range []string{"Registrant Address", "Registrant Addr", "Registrant Postal Address"} {
		body := "Domain Name: example.us\nRegistrant Name: Alex Example\n" + label + ": One Street\n  Suite: 2\n  City: Do not split this text\nRegistrant Email: owner@example.test\nTerms of use:\nRegistrant Street: Footer must not become an address\n"
		r, err := parseWHOIS("example.us", body)
		c := r.RegistrantContact
		if err != nil || c == nil || c.Name != "Alex Example" || !reflect.DeepEqual(c.Address, []string{"One Street", "Suite: 2", "City: Do not split this text"}) || len(c.Street) != 0 || c.City != "" || c.StateProvince != "" || c.Country != "" {
			t.Fatalf("%s: %+v %v", label, c, err)
		}
	}
}

func TestWHOISScopedAndEmbeddedRegistrantAddresses(t *testing.T) {
	for _, body := range []string{
		"Domain Name: example.us\nRegistrant:\n  Name: Alex Example\n  Organization: Example Organization\n  Address: One Street\n    Building 2\n  City: Example City\n  Country: US\nAdministrative Contact:\n  Name: Wrong Admin\n  Address: Wrong Street\n  Email: wrong@example.test\n",
		"Domain Name: example.us\nregistrant: OWNER-1\nadmin-c: ADMIN-1\n\nnic-hdl: OWNER-1\npersonname: Alex Example\norganization: Example Organization\naddress: One Street\n  Building 2\ncity: Example City\ncountry: US\n\nnic-hdl: ADMIN-1\npersonname: Wrong Admin\naddress: Wrong Street\ne-mail: wrong@example.test\n",
	} {
		r, err := parseWHOIS("example.us", body)
		c := r.RegistrantContact
		if err != nil || c == nil || c.Name != "Alex Example" || c.Organization != "Example Organization" || c.City != "Example City" || c.Country != "US" || !reflect.DeepEqual(c.Address, []string{"One Street", "Building 2"}) || len(c.Emails) != 0 {
			t.Fatalf("contact %+v, error %v", c, err)
		}
	}
}

func TestWHOISRedactedFieldsAndDetailedFieldBudget(t *testing.T) {
	body := "Domain Name: example.us\nRegistrant Name: REDACTED FOR PRIVACY\nRegistrant Organization: Example Organization\nRegistrant Street: REDACTED FOR PRIVACY\nRegistrant City: Public City\nRegistrant Phone: REDACTED FOR PRIVACY\nRegistrant Fax: +1.5550199\nRegistrant Email: REDACTED FOR PRIVACY\n"
	r, err := parseWHOIS("example.us", body)
	if err != nil || r.RegistrantContact == nil || r.RegistrantContact.Name != "" || r.RegistrantContact.Organization != "Example Organization" || len(r.RegistrantContact.Street) != 0 || len(r.RegistrantContact.Phones) != 0 || len(r.RegistrantContact.Emails) != 0 {
		t.Fatal(r.RegistrantContact, err)
	}
	for i := 0; i < 257; i++ {
		body += fmt.Sprintf("Registrant Street: Street %d\n", i)
	}
	if _, err := parseWHOIS("example.us", body); err == nil {
		t.Fatal("unbounded registrant address")
	}
}

const detailedRDAPFixture = `{"objectClassName":"domain","ldhName":"example.us","entities":[
 {"roles":["administrative"],"vcardArray":["vcard",[["fn",{},"text","Wrong Admin"],["adr",{},"text",["","","Wrong Street","Wrong City","","",""]]]]},
 {"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Alex Example"],["org",{},"text","Example City Authority"],["adr",{},"text",["","","400 Example Street","Example City","IL","60540","US"]],["tel",{"type":["work","voice"]},"uri","tel:+1.5550100"],["tel",{"type":"fax"},"text","+1.5550199"],["email",{},"text","owner@example.test"]]]}
]}`

func TestRDAPRegistrantNameOrganizationAndAddress(t *testing.T) {
	r, err := parseRDAP("example.us", []byte(detailedRDAPFixture))
	want := &RegistrantContact{Name: "Alex Example", Organization: "Example City Authority", Street: []string{"400 Example Street"}, City: "Example City", StateProvince: "IL", PostalCode: "60540", Country: "US", Phones: []string{"+1.5550100"}, Emails: []string{"owner@example.test"}}
	if err != nil || !reflect.DeepEqual(r.RegistrantContact, want) {
		t.Fatalf("%+v %v", r.RegistrantContact, err)
	}
	for _, domain := range []string{"example.gov", "example.edu", "zz"} {
		r, err := parseRDAP(domain, []byte(strings.ReplaceAll(detailedRDAPFixture, "example.us", domain)))
		if err != nil || r.RegistrantContact != nil {
			t.Fatal("dedicated layout changed", domain, r, err)
		}
	}
}

func TestRDAPWholeRegistrantAddressDoesNotInferComponents(t *testing.T) {
	body := `{"objectClassName":"domain","ldhName":"example.us","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","REDACTED FOR PRIVACY"],["org",{},"text","Example Organization"],["adr",{"label":"REDACTED FOR PRIVACY\nOne Street\nCity: Example City"},"text",["","","","","","",""]]]]}]}`
	r, err := parseRDAP("example.us", []byte(body))
	c := r.RegistrantContact
	if err != nil || c == nil || c.Name != "" || c.Organization != "Example Organization" || !reflect.DeepEqual(c.Address, []string{"One Street", "City: Example City"}) || c.City != "" || c.Country != "" {
		t.Fatal(c, err)
	}
}

func TestGenericRDAPFailureReturnsDetailedRegistryWHOISOnly(t *testing.T) {
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
	seedBootstrap(t, c, `[[["test"],["https://rdap.registry.test/"]]]`)
	seedFallbackTLD(t, c, "test", "generic", "active", "whois.registry.test")
	text := strings.ReplaceAll(detailedWHOISFixture, "example.us", "example.test") + "Registrar WHOIS Server: registrar.test\n"
	calls := fallbackWHOIS(t, c, "example.test", "whois.registry.test", text)
	r, err := c.Lookup(context.Background(), "example.test")
	if err != nil || *calls != 1 || r.RegistrantContact == nil || r.RegistrantContact.Name != "Alex Example" {
		t.Fatal(r, err, *calls)
	}
}
