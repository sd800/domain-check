package output

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"github.com/sd800/domain-check/internal/registry"
)

func TestEveryRemoteFieldBlocksTerminalInjection(t *testing.T) {
	payload := "prefix\x1b]52;c;YXR0YWNr\x07\x1b[2J\x1b]8;;https://evil.test\x1b\\\r\b\x9b\u202e\u2066suffix"
	r := registry.Record{Domain: "example.com", Registration: payload, Expiration: payload, Changed: payload, Statuses: []string{payload}, Registrar: payload, Registrant: payload, RegistrantEmail: []string{payload}, Nameservers: []string{payload}, DNSSEC: payload}
	r.RegistrantDetails = []string{payload, payload + "\n" + payload}
	r.RegistrantAddress = []string{payload}
	r.RegistrantCity, r.RegistrantState = payload, payload
	r.RegistrantPostalCode, r.RegistrantCountry = payload, payload
	tld := r
	tld.Domain, tld.TLD = "zz", true
	tld.RegistrantAddress, tld.RegistrationURL = []string{payload, payload}, payload
	tld.Admin, tld.AdminOrganization, tld.AdminRole = payload, payload, payload
	tld.AdminPhones, tld.AdminEmails = []string{payload}, []string{payload}
	detailed := r
	detailed.RegistrantContact = &registry.RegistrantContact{Name: payload, Organization: payload, Address: []string{payload}, Street: []string{payload}, City: payload, StateProvince: payload, PostalCode: payload, Country: payload, Phones: []string{payload}, Emails: []string{payload}}
	for _, r := range []registry.Record{r, tld, detailed} {
		for _, lang := range []Language{English, Chinese} {
			var b bytes.Buffer
			if err := Format(&b, r, lang, 80); err != nil {
				t.Fatal(err)
			}
			for _, ch := range b.String() {
				if ch != '\n' && (unicode.IsControl(ch) || unicode.In(ch, unicode.Cf)) {
					t.Fatalf("executable terminal control survived: %U", ch)
				}
			}
			if !strings.Contains(b.String(), "suffix") {
				t.Fatal("sanitization discarded ordinary field data")
			}
		}
	}
}
