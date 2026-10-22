package registry

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

const allocationFixture = `{
 "objectClassName":"domain","ldhName":"zz","status":["active"],
 "events":[{"eventAction":"registration","eventDate":"1994-04-08T04:00:00Z"},{"eventAction":"last changed","eventDate":"2026-10-22"}],
 "links":[{"rel":"related","title":"Registration URL","href":"http://registry.example.test/register"},{"rel":"alternate","href":"https://registrar.test/domain/zz"}],
 "entities":[
  {"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Registry Contact"],["org",{},"text","Example Registry"],["adr",{"label":"One Registry Road\nExample City 12345\nExample Country"},"text",["","","","","","",""]]]],"events":[{"eventAction":"last changed","eventDate":"2000-01-01"}]},
  {"roles":["technical"],"vcardArray":["vcard",[["fn",{},"text","Wrong Contact"],["role",{},"text","Admin"],["email",{},"text","wrong@example.test"]]]},
  {"roles":["administrative"],"vcardArray":["vcard",[["fn",{},"text","Alex Example"],["org",{},"text","Example Organization"],["role",{},"text","Operations Lead"],["tel",{"type":["work","voice"]},"text","+1-555-0100"],["tel",{"type":["work","fax"]},"text","+1-555-0101"],["email",{},"text","admin@example.test"]]]}
 ],
 "nameservers":[{"ldhName":"NS1.EXAMPLE.TEST."},{"ldhName":"ns2.example.test"}],"secureDNS":{"delegationSigned":true}
}`

func TestTLDQueryGoesOnlyToIANAAndDoesNotCacheResults(t *testing.T) {
	calls := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://rdap.iana.org/domain/zz" {
			t.Fatalf("TLD query left IANA: %s", r.URL)
		}
		return reply(r, 200, allocationFixture), nil
	})
	r, err := c.Lookup(context.Background(), ".ZZ")
	if err != nil || calls != 1 || !r.TLD || r.Domain != "zz" {
		t.Fatal(r, err, calls)
	}
	if r.Registrant != "Example Registry" || r.Admin != "Alex Example" || r.AdminOrganization != "Example Organization" || r.AdminRole != "Operations Lead" {
		t.Fatal("RDAP role and jCard role were confused", r)
	}
	if !reflect.DeepEqual(r.RegistrantAddress, []string{"One Registry Road", "Example City 12345", "Example Country"}) || !reflect.DeepEqual(r.AdminPhones, []string{"+1-555-0100", "+1-555-0101"}) || !reflect.DeepEqual(r.AdminEmails, []string{"admin@example.test"}) || len(r.RegistrantEmail) != 0 {
		t.Fatal("contact fields were mixed", r)
	}
	if r.RegistrationURL != "http://registry.example.test/register" || r.Registration != "1994-04-08 04:00:00 UTC" || r.Changed != "2026-10-22" || r.DNSSEC != "signed" {
		t.Fatal(r)
	}
	if files, err := os.ReadDir(c.CacheDir); err != nil || len(files) != 0 {
		t.Fatal("TLD lookup saved data", files, err)
	}
}

func TestTLDAbsentInactiveAndServiceFailuresRemainDistinct(t *testing.T) {
	for _, tc := range []struct {
		name, body, code string
		status           int
		absent, inactive bool
	}{
		{"absent", `{"errorCode":404}`, "", 404, true, false},
		{"inactive", `{"objectClassName":"domain","ldhName":"zz","status":["inactive"]}`, "", 200, false, true},
		{"reserved", `{"objectClassName":"domain","ldhName":"zz","status":["reserved"]}`, "", 200, false, false},
		{"denied", `{}`, "access-denied", 403, false, false},
		{"rate", `{}`, "rate-limited", 429, false, false},
		{"server", `{}`, "rdap-failed", 503, false, false},
		{"mismatch", `{"objectClassName":"domain","ldhName":"example.zz"}`, "invalid-response", 200, false, false},
		{"bad body", `{"errorCode":404}`, "invalid-response", 200, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.String() != "https://rdap.iana.org/domain/zz" {
					t.Fatal(req.URL)
				}
				return reply(req, tc.status, tc.body), nil
			})
			r, err := c.Lookup(context.Background(), "zz")
			if calls != 1 {
				t.Fatal("TLD query followed another service", calls)
			}
			if tc.code != "" {
				var query *QueryError
				if !errors.As(err, &query) || query.Code != tc.code {
					t.Fatal(r, err)
				}
				return
			}
			if err != nil || !r.TLD || (r.Result == NoRecord) != tc.absent {
				t.Fatal(r, err)
			}
			if tc.inactive && (r.Result != "" || len(r.Statuses) != 1 || r.Statuses[0] != "inactive") {
				t.Fatal("inactive TLD became an availability claim", r)
			}
		})
	}
}

func TestTLDRedirectAndUnicodeIdentityAreConstrained(t *testing.T) {
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		response := reply(r, 302, "")
		response.Header.Set("Location", "https://registrar.test/domain/zz")
		return response, nil
	})
	if _, err := c.Lookup(context.Background(), "zz"); err == nil || !strings.Contains(err.Error(), "redirect-blocked") {
		t.Fatal(err)
	}
	for _, input := range []string{"中国", ".中国", "xn--fiqs8s"} {
		c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://rdap.iana.org/domain/xn--fiqs8s" {
				t.Fatal(r.URL)
			}
			return reply(r, 200, `{"objectClassName":"domain","ldhName":"xn--fiqs8s","unicodeName":"中国"}`), nil
		})
		r, err := c.Lookup(context.Background(), input)
		if err != nil || !r.TLD || UnicodeDomain(r.Domain) != "中国" {
			t.Fatal(r, err)
		}
	}
	if _, err := parseRDAP("xn--fiqs8s", []byte(`{"objectClassName":"domain","ldhName":"xn--fiqs8s","unicodeName":"different"}`)); err == nil {
		t.Fatal("conflicting Unicode identity accepted")
	}
}

func TestTLDCardRoleDoesNotSelectAdminAndStructuredAddressWorks(t *testing.T) {
	body := strings.Replace(allocationFixture, `"roles":["administrative"]`, `"roles":["technical"]`, 1)
	r, err := parseRDAP("zz", []byte(body))
	if err != nil || r.Admin != "" || r.AdminRole != "" || len(r.AdminEmails) != 0 {
		t.Fatal("jCard role was used to select an administrator", r, err)
	}
	lines := contactAddress([]byte(`{}`), []any{"", "", "One Street", "City", "Region", "12345", "Country"})
	if !reflect.DeepEqual(lines, []string{"One Street", "City Region 12345", "Country"}) {
		t.Fatal(lines)
	}
}
