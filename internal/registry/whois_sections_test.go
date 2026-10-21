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
