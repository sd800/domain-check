package registry

import (
	"fmt"
	"strings"
	"testing"
)

func TestAdditionalRegistryFieldFormats(t *testing.T) {
	for _, test := range []struct {
		domain, body string
		check        func(*testing.T, Record)
	}{
		{"example.lt", "Domain: example.lt\nStatus: registered\nRegistered: 2001-02-03\nExpires: 2027-02-03\nRegistrar: Example Registrar\nRegistrar email: registrar@example.test\n%\nContact organization: Example University\nContact email: owner@example.test\n%\nNameserver: ns1.example.test\nDS: 12345 13 2 ABCDEF0123456789\n", func(t *testing.T, r Record) {
			if r.Registrant != "Example University" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" || r.Registrar != "Example Registrar" || r.Registration != "2001-02-03" || len(r.Nameservers) != 1 || r.DNSSEC != "signedDelegation" {
				t.Fatalf("%+v", r)
			}
		}},
		{"example.tm", "Domain: example.tm\nStatus: Live\nExpiry: 2027-02-03\nOwner Name: Domain Administrator\nOwner OrgName: Example University\nOwner Addr: Example Road\nNS 1: ns1.example.test\n", func(t *testing.T, r Record) {
			if r.Registrant != "Example University" || r.Expiration != "2027-02-03" || len(r.Nameservers) != 1 {
				t.Fatalf("%+v", r)
			}
		}},
		{"example.lu", "domainname: example.lu\ndomaintype: ACTIVE\nownertype: ORGANISATION\norg-country: LU\nregistrar-name: Example Registrar\nregistrar-email: registrar@example.test\nnserver: ns1.example.test\n", func(t *testing.T, r Record) {
			if strings.Join(r.Statuses, ",") != "ACTIVE" || r.Registrant != "" || len(r.RegistrantEmail) != 0 || r.Registrar != "Example Registrar" {
				t.Fatalf("%+v", r)
			}
		}},
		{"example.ug", "Domain name: example.ug\nRegistered On: 2001-02-03\nExpires On: 2027-02-03\nRegistrant Contact Information:\nRegistrant Name: Example Owner\nRegistrant Email: owner@example.test\nAdministrative Contact Information:\nEmail: admin@example.test\nTechnical Contact Information:\nEmail: tech@example.test\nInformation Last Updated: 2026-03-04T12:00:00.000+03:00\n", func(t *testing.T, r Record) {
			if r.Changed != "2026-03-04 09:00:00 UTC" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" {
				t.Fatalf("%+v", r)
			}
		}},
	} {
		t.Run(test.domain, func(t *testing.T) {
			r, err := parseWHOIS(test.domain, test.body)
			if err != nil {
				t.Fatal(err)
			}
			test.check(t, r)
		})
	}
}

func TestProtocolRestrictionNotices(t *testing.T) {
	for _, test := range []struct {
		body string
		want Result
	}{
		{"The Domain Name is not Available", Unavailable},
		{"stop-list: Domain can not be registered.", NotAllowed},
		{"Domain Name: example.my\n>>> This name is not available for registration:\n>>> IANA Reserved\n>>> Last update of WHOIS database: 2026-10-21T00:00:00Z\n", Reserved},
		{"Domain Name: example.my\n>>> This name is not available for registration:\n>>> Domain name matches a restricted word\n>>> Last update of WHOIS database: 2026-10-21T00:00:00Z\n", NotAllowed},
		{"Domain Name: example.my\n>>> This name is not available for registration:\n>>> Last update of WHOIS database: 2026-10-21T00:00:00Z\n", Unavailable},
	} {
		r, err := parseWHOIS("example.my", test.body)
		if err != nil || r.Result != test.want {
			t.Fatalf("%q: %+v, %v", test.body, r, err)
		}
	}
}

func TestWHOISQueryFailuresRemainErrors(t *testing.T) {
	for _, body := range []string{"Malformed request.", "%ERROR:102: Invalid domain name", "-7: %Invalid pattern", "object is blocked\n", "Wrong top level domain name in query", "Erreur\tSyntax [ Domain ]", "whois.nic.bo solo acepta consultas con dominios .bo", "Domain Name: example.my\n>>> This name is not available for registration:\n>>> Tld not supported by this registry interface\n"} {
		r, err := parseWHOIS("example.my", body)
		q, ok := err.(*QueryError)
		if !ok || q.Code != "whois-failed" || r.Result != "" {
			t.Errorf("%q: %+v, %v", body, r, err)
		}
	}
	// A registry's examples and footer warnings are not query results.
	r, err := parseWHOIS("example.my", "Domain Name: example.my\nCreation Date: 2001-02-03\n>>> Last update of WHOIS database: 2026-10-21T00:00:00Z\n>>> IANA Reserved\n")
	if err != nil || r.Result != "" || r.Registration != "2001-02-03" {
		t.Fatalf("footer changed a record: %+v, %v", r, err)
	}
}

func TestSerbianIDNDateConventions(t *testing.T) {
	domain := "xn--e1afmkfd.xn--90a3ac"
	body := fmt.Sprintf("Domain name: %s\nRegistration date: 03.02.2001 04:05:06\nModification date: 05.04.2026 06:07:08\nExpiration date: 03.02.2027 04:05:06\nRegistrant: Example University\n", domain)
	r, err := parseWHOIS(domain, body)
	if err != nil || r.Registration != "2001-02-03 04:05:06" || r.Changed != "2026-04-05 06:07:08" || r.Expiration != "2027-02-03 04:05:06" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestBulgarianInvalidNamesAreNotRecords(t *testing.T) {
	for _, domain := range []string{"example.bg", "example.xn--90ae"} {
		r, err := parseWHOIS(domain, "DOMAIN NAME: "+domain+"\nregistration status: invalid\n")
		if err != nil || r.Result != NotAllowed {
			t.Fatalf("%+v, %v", r, err)
		}
	}
	r, err := parseWHOIS("example.com", "Domain Name: example.com\nDomain Status: invalid\n")
	if err != nil || r.Result != "" || strings.Join(r.Statuses, ",") != "invalid" {
		t.Fatalf("registry rule leaked to another format: %+v, %v", r, err)
	}
}

func TestUkrainianIDNExplicitIdentityFields(t *testing.T) {
	domain, err := NormalizeDomain("приклад.укр")
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("Domain Name (PUNYCODE): %s\nDomain name (UTF8): приклад.укр\nUpdated Date: 19-Sep-2026 13:03:43 UTC\nCreation Date: 20-Mar-2013 14:03:43 UTC\nExpiration Date: 20-Aug-2027 13:03:43 UTC\nRegistry Status: ok\nRegistrar: Example Registrar\nRegistrant Name (Organization): Example University\nRegistrant Email: owner@example.test\nAdmin Email: admin@example.test\nTech Email: tech@example.test\nDomain servers in listed order:\n ns1.example.test 192.0.2.1\n ns2.example.test 2001:db8::1\n", domain)
	r, err := parseWHOIS(domain, body)
	if err != nil || r.Registration != "2013-03-20 14:03:43 UTC" || r.Expiration != "2027-08-20 13:03:43 UTC" || r.Changed != "2026-09-19 13:03:43 UTC" || r.Registrant != "Example University" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" || strings.Join(r.Statuses, ",") != "ok" || len(r.Nameservers) != 2 {
		t.Fatalf("%+v, %v", r, err)
	}
	if _, err := parseWHOIS(domain, strings.Replace(body, "приклад.укр", "інший.укр", 1)); err == nil {
		t.Fatal("contradictory Unicode identity accepted")
	}
}

func TestLithuanianContactAliasesDoNotLeak(t *testing.T) {
	r, err := parseWHOIS("example.com", "Domain: example.com\nContact organization: Technical Operator\nContact email: tech@example.test\n")
	if err != nil || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
		t.Fatalf("generic contact data substituted for owner: %+v, %v", r, err)
	}
}

func TestNamedAbsenceMustMatchQueriedDomain(t *testing.T) {
	for _, template := range []string{"No matching record for %s.", "No data found for domain: %s.", "No record found for %s.", "No match for domain %s."} {
		r, err := parseWHOIS("example.tm", fmt.Sprintf(template, "example.tm"))
		if err != nil || r.Result != NoRecord {
			t.Errorf("matching result lost: %+v, %v", r, err)
		}
		if r, err := parseWHOIS("example.tm", fmt.Sprintf(template, "other.tm")); err == nil || r.Result != "" {
			t.Errorf("different domain implied availability: %+v, %v", r, err)
		}
	}
}

func TestAdditionalIDNRegistryFormats(t *testing.T) {
	for _, test := range []struct {
		domain, body string
		check        func(Record) bool
	}{
		{"example.xn--d1alf", "domain: example.xn--d1alf\nregistrant: HOLDER1\ncreated: 03.02.2001 04:05:06\nexpire: 03.02.2027\n", func(r Record) bool {
			return r.Registration == "2001-02-03 04:05:06" && r.Expiration == "2027-02-03" && r.Registrant == ""
		}},
		{"example.xn--4dbrk0ce", "domain: example.xn--4dbrk0ce\nvalidity: 03-02-2027\nchanged: contact AT example.test 20260405 (Changed)\ne-mail: owner@example.test\nadmin-email: admin@example.test\n", func(r Record) bool {
			return r.Expiration == "2027-02-03" && r.Changed == "2026-04-05" && strings.Join(r.RegistrantEmail, ",") == "owner@example.test"
		}},
		{"example.xn--90ais", "Domain: example.xn--90ais\nEmail: owner@example.test\nTechnical Contact:\nEmail: tech@example.test\n", func(r Record) bool { return strings.Join(r.RegistrantEmail, ",") == "owner@example.test" }},
		{"example.xn--mgba3a4f16a", "domain: example.xn--mgba3a4f16a\nregistrant: HOLDER1\nnserver: ns1.example.test\n", func(r Record) bool { return r.Registrant == "" && len(r.Nameservers) == 1 }},
	} {
		r, err := parseWHOIS(test.domain, test.body)
		if err != nil || !test.check(r) {
			t.Errorf("%s: %+v, %v", test.domain, r, err)
		}
	}
}
