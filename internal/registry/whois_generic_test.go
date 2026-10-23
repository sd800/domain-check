package registry

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestGenericThinRegistryRecords(t *testing.T) {
	for _, tld := range []string{"com", "net", "name"} {
		domain := "example." + tld
		body := "Domain Name: " + domain + "\nRegistry Domain ID: 1234\nRegistrar: Example Registrar\nRegistrar WHOIS Server: whois.registrar.test\nRegistrar Abuse Contact Email: abuse@registrar.test\nDomain Status: clientTransferProhibited https://icann.org/epp#clientTransferProhibited\n"
		want := Record{Domain: domain, Registrar: "Example Registrar", Statuses: []string{"clientTransferProhibited"}}
		if tld != "name" {
			body += "Creation Date: 2000-01-02T03:04:05Z\nRegistry Expiry Date: 2030-01-02T03:04:05Z\nUpdated Date: 2026-01-02T03:04:05Z\nName Server: NS1.EXAMPLE.TEST\nDNSSEC DS Data: 12345 13 2 ABCDEF0123456789\n"
			want.Registration, want.Expiration, want.Changed = "2000-01-02 03:04:05 UTC", "2030-01-02 03:04:05 UTC", "2026-01-02 03:04:05 UTC"
			want.Nameservers, want.DNSSEC = []string{"ns1.example.test"}, "signedDelegation"
		}
		body += ">>> Last update of whois database: 2099-12-31T23:59:59Z <<<\nRegistrant Email: wrong@footer.test\n"
		r, err := parseWHOIS(domain, body)
		if err != nil || !reflect.DeepEqual(r, want) {
			t.Fatalf("%s: %+v, %v", tld, r, err)
		}
	}
}

func TestGenericRegistryContactsAndRedactions(t *testing.T) {
	for _, tld := range []string{"art", "bank", "museum", "samsung", "wang", "xn--d1acj3b"} {
		text := strings.ReplaceAll(detailedWHOISFixture, "example.us", "example."+tld)
		text = strings.ReplaceAll(text, "Alex Example", "Example City Authority")
		text += "Registry Admin ID: ADMIN-1\nAdmin Organization: Wrong Organization\nDNSSEC: unsigned\n"
		r, err := parseWHOIS("example."+tld, text)
		c := r.RegistrantContact
		if err != nil || c == nil || c.Name != "Example City Authority" || c.Organization != "Example City Authority" || c.City != "Example City" || len(c.Phones) != 1 || len(c.Emails) != 1 || c.Emails[0] != "owner@example.test" || r.DNSSEC != "unsigned" {
			t.Fatal(tld, c, r.DNSSEC, err)
		}
	}
	text := "Domain Name: example.wang\nRegistrant Name: REDACTED FOR PRIVACY\nRegistrant Organization: Example Organization\nRegistrant Street: REDACTED FOR PRIVACY\nRegistrant Country: CN\nRegistrant Email: Please query the RDDS service of the Registrar of Record\nAdmin Email: wrong@admin.test\nName Server: ns1.example.test\n"
	r, err := parseWHOIS("example.wang", text)
	c := r.RegistrantContact
	if err != nil || c == nil || c.Name != "" || c.Organization != "Example Organization" || c.Country != "CN" || len(c.Street) != 0 || len(c.Emails) != 0 {
		t.Fatal(c, err)
	}
}

func TestIANAIntOwnerAddressStopsBeforeOtherContacts(t *testing.T) {
	text := "% IANA WHOIS server\ndomain: EXAMPLE.INT\norganisation: Example International Organization\naddress: One Street\naddress: Example City\naddress: Example Country\n\ncontact: administrative\nname: Wrong Admin\naddress: Wrong Address\ne-mail: wrong@admin.test\n\ncontact: technical\nname: Wrong Tech\naddress: Wrong Technical Address\n\nnserver: NS1.EXAMPLE.TEST 192.0.2.1\ncreated: 2000-01-02\nchanged: 2026-01-02\nsource: IANA\n"
	r, err := parseWHOIS("example.int", text)
	c := r.RegistrantContact
	if err != nil || r.Registrant != "Example International Organization" || c == nil || c.Name != "" || c.Organization != r.Registrant || !reflect.DeepEqual(c.Address, []string{"One Street", "Example City", "Example Country"}) || len(c.Emails) != 0 || len(c.Phones) != 0 || r.Registration != "2000-01-02" || r.Changed != "2026-01-02" {
		t.Fatal(r, c, err)
	}
}

func TestGenericWHOISAbsenceDialects(t *testing.T) {
	for _, body := range []string{
		"Available\n\nNo information was found matching that query.\n>>> Last update of WHOIS database: 2099-12-31T23:59:59Z <<<\n",
		"The queried object does not exist\n>>> Last update of Whois database: 2099-12-31T23:59:59Z <<<\n",
	} {
		for _, tld := range []string{"africa", "capetown", "cologne", "berlin", "hamburg", "ikano"} {
			domain := "example-unregistered." + tld
			r, err := parseWHOIS(domain, body)
			if err != nil || !reflect.DeepEqual(r, Record{Domain: domain, Result: NoRecord}) {
				t.Fatal(domain, r, err)
			}
		}
	}
	if _, err := parseWHOIS("example.africa", "Available\n"); err == nil {
		t.Fatal("ambiguous word became an availability claim")
	}
}

func TestGenericWHOISServiceNoticesAreFailures(t *testing.T) {
	for _, body := range []string{
		"Notice: Effective May 1, 2026, the WHOIS service has been retired in accordance with the registry policy. Queries now use RDAP at https://registry.test/rdap/",
		"TLD is not supported.",
		"%error 350 Invalid Query Syntax",
		"Error: ratelimit exceeded",
		"Queried interval is too short.",
		"Your access is too fast,please try again later.",
	} {
		for _, prefix := range []string{"", "% "} {
			r, err := parseWHOIS("example.test", prefix+body+"\n")
			if q, ok := err.(*QueryError); !ok || q.Code != "whois-failed" || r.Result != "" {
				t.Fatal(body, r, err)
			}
		}
	}
}

func TestGenericFallbackKeepsServiceRetirementAnError(t *testing.T) {
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
	seedBootstrap(t, c, `[[["test"],["https://rdap.registry.test/"]]]`)
	seedFallbackTLD(t, c, "test", "generic", "active", "whois.registry.test")
	calls := fallbackWHOIS(t, c, "example.test", "whois.registry.test", "Notice: Effective May 1, 2026, the WHOIS service has been retired in accordance with the registry policy. Use https://other.test/rdap/\n")
	r, err := c.Lookup(context.Background(), "example.test")
	if q, ok := err.(*QueryError); !ok || q.Code != "fallback-failed" || r.Result != "" || *calls != 1 {
		t.Fatal(r, err, *calls)
	}
}
