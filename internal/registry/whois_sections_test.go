package registry

import (
	"strings"
	"testing"
)

func TestIMContactLabelsAndExpiry(t *testing.T) {
	for _, test := range []struct{ name, response, owner string }{
		{"empty names", `Domain Name: example.im
Domain Managers
Name:
Address
Name:
Address
Expiry Date: 21/10/2030 00:59:59
Name Server: ns1.example.test`, ""},
		{"redacted names", `Domain Name: example.im
Domain Managers
Name: REDACTED FOR PRIVACY
Address
REDACTED FOR PRIVACY
Name: REDACTED FOR PRIVACY
Address
Expiry Date: 21/10/2030 00:59:59
Name Server: ns1.example.test`, ""},
		{"first named manager", `Domain Name: example.im
Domain Managers
Name
Example Owner
Address
1 Example Street
Example City
Name: Other Manager
Address
2 Other Street
Expiry Date: 21/10/2030 00:59:59
Name Server: ns1.example.test`, "Example Owner"},
		{"hidden owner is not another manager", `Domain Name: example.im
Domain Managers
Name: REDACTED FOR PRIVACY
Address
Name: Other Manager
Address
Expiry Date: 21/10/2030 00:59:59
Name Server: ns1.example.test`, ""},
		{"standalone expiry label", `Domain Name: example.im
Domain Managers
Name:
Address
Expiry Date
21/10/2030 00:59:59
Name Server
ns1.example.test`, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, err := parseWHOIS("example.im", test.response)
			if err != nil || r.Registrant != test.owner || r.Expiration != "2030-10-21 00:59:59" || strings.Join(r.Nameservers, ",") != "ns1.example.test" {
				t.Fatalf("%+v, %v", r, err)
			}
		})
	}
}

func TestWHOISEmptyNameNeverConsumesContactLabels(t *testing.T) {
	for _, tld := range []string{"im", "edu", "am", "bn", "sm", "ch", "li", "gg", "je"} {
		for _, label := range []string{"Address", "Street", "City", "Country", "Postal Code", "Phone", "Fax"} {
			domain := "example." + tld
			for _, name := range []string{"Name", "Name:", "Name: REDACTED FOR PRIVACY"} {
				body := "Domain Name: " + domain + "\nRegistrant:\n" + name + "\n" + label + "\nSome address value\nName Servers:\nns1.example.test\n"
				r, err := parseWHOIS(domain, body)
				if err != nil || r.Registrant != "" || len(r.RegistrantEmail) != 0 || len(r.Nameservers) != 1 {
					t.Fatalf("%s, %s, %s: %+v, %v", tld, name, label, r, err)
				}
			}
		}
	}
}

func TestWHOISRedactedInlineValueNeverConsumesNextLine(t *testing.T) {
	for _, field := range []string{"Registrant", "Registrant Name", "Registrant Organization", "Registrar", "Creation Date", "Expiry Date", "Updated Date", "DNSSEC"} {
		r, err := parseWHOIS("example.im", "Domain Name: example.im\n"+field+": REDACTED FOR PRIVACY\nThis is unrelated text\nName Server: ns1.example.test\n")
		if err != nil || r.Registrant != "" || r.Registrar != "" || r.Registration != "" || r.Expiration != "" || r.Changed != "" || r.DNSSEC != "" {
			t.Fatalf("%s consumed prose after a redacted value: %+v, %v", field, r, err)
		}
	}
}

func TestWHOISContactDatesStayScopedWhileDomainExpiryResumes(t *testing.T) {
	r, err := parseWHOIS("example.it", `Domain: example.it
Created: 2000-10-21
Last Update: 2026-10-20
Registrant
Name: Example Owner
Created: 1990-01-01
Last Update: 2026-10-21
Admin Contact
Name: Wrong Admin
Created: 1991-01-01
Last Update: 2026-10-21
Registry Expiry Date: 2030-10-21
Nameservers
ns1.example.test`)
	if err != nil || r.Registrant != "Example Owner" || r.Registration != "2000-10-21" || r.Changed != "2026-10-20" || r.Expiration != "2030-10-21" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISManagerAndOwnerSectionsRemainSeparate(t *testing.T) {
	for _, test := range []struct {
		name, ownerSection, ownerValue, wantOwner, wantEmail string
	}{
		{"disclosed owner", "Domain Owners / Registrant", "Example Owner", "Example Owner", "owner@example.test"},
		{"redacted owner", "Domain Owners / Registrant", "Redacted", "", ""},
		{"unspaced owner heading", "Domain Owners/Registrant", "Example Owner", "Example Owner", "owner@example.test"},
		{"singular owner heading", "Domain Owner / Registrant", "Example Owner", "Example Owner", "owner@example.test"},
	} {
		t.Run(test.name, func(t *testing.T) {
			email := "owner@example.test"
			if test.ownerValue == "Redacted" {
				email = "Redacted"
			}
			body := `This information has been redacted. Please contact support@example.test with queries.
Domain Name: example.im
Domain Managers
Name: Example Manager
Organization: Example Management Company
Email: manager@example.test
Address
1 Management Street
` + test.ownerSection + "\nName: " + test.ownerValue + "\nEmail: " + email + `
Address
Redacted
Administrative Contact
Name: Wrong Admin
Email: admin@example.test
Billing Contact
Name: Wrong Billing Contact
Email: billing@example.test
Technical Contact
Name: Wrong Technical Contact
Email: tech@example.test
Domain Details
Created: 2000-10-21
Last Updated: 2026-10-20
Expiry Date: 05/11/2030 00:59:44
Name Server: ns1.example.test.
Name Server: ns2.example.test.
`
			r, err := parseWHOIS("example.im", body)
			if err != nil || r.Registrant != test.wantOwner || strings.Join(r.RegistrantEmail, ",") != test.wantEmail || r.Registration != "2000-10-21" || r.Changed != "2026-10-20" || r.Expiration != "2030-11-05 00:59:44" || strings.Join(r.Nameservers, ",") != "ns1.example.test,ns2.example.test" {
				t.Fatalf("%+v, %v", r, err)
			}
		})
	}
}

func TestWHOISRegistryOwnedDomainDoesNotInventOrdinaryFields(t *testing.T) {
	// A registry-owned name can have a substantially smaller WHOIS record.
	// Its missing dates and nameservers must stay absent.
	r, err := parseWHOIS("example.im", `Domain Name: example.im
Domain Managers
Name
Example Registry
Address
Example Operator Ltd
1 Registry Street
Example City
`)
	if err != nil || r.Registrant != "Example Registry" || r.Registration != "" || r.Changed != "" || r.Expiration != "" || len(r.Nameservers) != 0 || r.Registrar != "" || len(r.RegistrantEmail) != 0 {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISTLDReferralPreambleDoesNotBecomeDomainData(t *testing.T) {
	r, err := parseWHOIS("example.im", `% IANA WHOIS server
refer: whois.registry.test
domain: IM
organisation: Wrong TLD Organization
nserver: wrong.tld-ns.test
created: 1996-09-11
changed: 2018-11-24
source: IANA

# whois.registry.test
Domain Name: example.im
Domain Managers
Name
Example Registry
Address
Example Operator Ltd
`)
	if err != nil || r.Registrant != "Example Registry" || r.Registration != "" || r.Changed != "" || len(r.Nameservers) != 0 {
		t.Fatalf("TLD metadata leaked into the domain record: %+v, %v", r, err)
	}
}

func TestWHOISCommentedAndRegistrySpecificNegativeReplies(t *testing.T) {
	for body, want := range map[string]Result{
		"%ERROR:101: no entries found\n% No entries found.":   NoRecord,
		"%ERROR: No entries found":                            NoRecord,
		"% No match for domain 'example.aw'.":                 NoRecord,
		">> No data found for domain : example.aw":            NoRecord,
		"example.aw is free":                                  NotRegistered,
		"This name is reserved by the Registry.":              Reserved,
		"Reserved Domain Name":                                Reserved,
		"Domain example.aw has been reserved by the Registry": Reserved,
	} {
		r, err := parseWHOIS("example.aw", body)
		if err != nil || r.Result != want {
			t.Errorf("%q: %+v, %v", body, r, err)
		}
	}
	for _, body := range []string{
		"other.aw is free",
		"Domain other.aw has been reserved by the Registry",
		"Requests of this client are not permitted. Please use the registry website.",
		"The domain you requested is not known in the database.\nThis TLD has no whois server.",
		"% Domains may be reserved by the registry.",
	} {
		if r, err := parseWHOIS("example.aw", body); err == nil || r.Result != "" {
			t.Errorf("failure or unrelated text became availability: %q: %+v, %v", body, r, err)
		}
	}
}

func TestWHOISImplicitDomainIdentityDoesNotRequireDelegation(t *testing.T) {
	r, err := parseWHOIS("example.cl", `Registrant name: Example University
Registrar name: Example Registrar
Creation date: 2000-10-21 01:02:03 CLST
Expiration date: 2030-10-21 01:02:03 CLST
`)
	if err != nil || r.Registrant != "Example University" || r.Registration != "2000-10-21 04:02:03 UTC" || r.Expiration != "2030-10-21 04:02:03 UTC" || len(r.Nameservers) != 0 {
		t.Fatalf("a record without nameservers was rejected: %+v, %v", r, err)
	}
}

func TestWHOISReferralFieldDoesNotHideFollowingDomainDates(t *testing.T) {
	for _, referral := range []string{"WHOIS Server:", "WHOIS Server: whois.registrar.test"} {
		body := "Domain Name: example.ws\n" + referral + `
Registrar URL: https://registrar.test/
Updated Date: 2026-10-20T12:34:56Z
Creation Date: 2000-10-21T01:02:03Z
Registrar Registration Expiration Date: 2030-10-21T01:02:03Z
Name Server: ns1.example.test
`
		r, err := parseWHOIS("example.ws", body)
		if err != nil || r.Registration != "2000-10-21 01:02:03 UTC" || r.Changed != "2026-10-20 12:34:56 UTC" || r.Expiration != "2030-10-21 01:02:03 UTC" {
			t.Fatalf("%s hid domain dates: %+v, %v", referral, r, err)
		}
	}
}

func TestWHOISParentRecordCannotFillChildDomain(t *testing.T) {
	r, err := parseWHOIS("child.ac.rs", `Domain name: ac.rs
Domain status: Active
Registration date: 2000-10-21
Expiration date: 2030-10-21
Registrant: Example Registry
Name Server: ns1.example.test
`)
	if err != nil || r.Domain != "child.ac.rs" || r.Result != NotProvided || r.Registration != "" || r.Registrant != "" || len(r.Nameservers) != 0 {
		t.Fatalf("parent record became a child-domain result: %+v, %v", r, err)
	}
}
