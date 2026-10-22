package registry

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const twRecord = `Domain Name: example.com.tw
   Domain Status: clientUpdateProhibited,clientTransferProhibited,clientDeleteProhibited
   Registrant:

      Example University
      REGISTRY-HANDLE owner@example.test
      TW
      (Redacted for privacy)

   Administrative Contact:
      Example Administrator admin@example.test

   Technical Contact:
      Example Technician technical@example.test

   Record expires on 2027-11-09 00:00:00 (UTC+8)
   Record created on 2000-08-29 10:22:50 (UTC+8)

   Domain servers in listed order:
      NS1.EXAMPLE.TEST 192.0.2.1
      NS2.EXAMPLE.TEST

Registration Service Provider: Example Registrar
Registration Service URL: https://registrar.test/
Registrar Abuse Contact Email: abuse@example.test
Registrar WHOIS Server: whois.registrar.test
`

func TestTWNICStandardizedFields(t *testing.T) {
	r, err := parseWHOIS("example.com.tw", twRecord)
	if err != nil || r.Registrant != "Example University" || !reflect.DeepEqual(r.RegistrantEmail, []string{"owner@example.test"}) || r.Registrar != "Example Registrar" || r.Registration != "2000-08-29 02:22:50 UTC" || r.Expiration != "2027-11-08 16:00:00 UTC" || r.Changed != "" || r.DNSSEC != "" || len(r.Nameservers) != 2 || len(r.Statuses) != 3 || r.Statuses[1] != "clientTransferProhibited" {
		t.Fatal(r, err)
	}
	redacted := strings.Replace(twRecord, "Example University\n      REGISTRY-HANDLE owner@example.test", "(Redacted for privacy)", 1)
	r, err = parseWHOIS("example.com.tw", redacted)
	if err != nil || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
		t.Fatal("redacted owner became a country/address or another contact", r, err)
	}
}

func TestTWUsesBulkIANADiscoveredWHOISWithoutRDAPOrReferrals(t *testing.T) {
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		t.Fatalf(".tw queried HTTP or RDAP: %s", req.URL)
		return nil, nil
	})
	seedBootstrap(t, c, `[[["tw"],["https://rdap.registry.test/tw/"]]]`)
	seedTLDs(t, c, map[string]string{"tw": "whois.twnic.net.tw"})
	var calls []string
	c.DialWHOIS = func(_ context.Context, _, address string) (net.Conn, error) {
		calls = append(calls, address)
		if address != "whois.twnic.net.tw:43" {
			t.Fatal("query left registry", address)
		}
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			query, _ := bufio.NewReader(server).ReadString('\n')
			if query != "example.com.tw\r\n" {
				t.Errorf("unexpected WHOIS query %q", query)
			}
			fmt.Fprint(server, twRecord)
		}()
		return client, nil
	}
	r, err := c.Lookup(context.Background(), "example.com.tw")
	if err != nil || r.Registrar != "Example Registrar" || len(calls) != 1 {
		t.Fatal(r, err, calls)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, "example.com.tw", "owner@example.test")
	c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("unreachable") }
	if _, err := c.Lookup(context.Background(), "example.com.tw"); err == nil {
		t.Fatal("WHOIS failure became an empty record")
	}
}

func TestTWOverrideDoesNotAffectTLDsOrOtherSuffixes(t *testing.T) {
	for input, endpoint := range map[string]string{"tw": "https://rdap.iana.org/domain/tw", "example.tw.com": "https://rdap.registry.test/domain/example.tw.com", "example.xn--kpry57d": "https://rdap.registry.test/domain/example.xn--kpry57d"} {
		c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != endpoint {
				t.Fatal(input, req.URL)
			}
			return reply(req, 200, fmt.Sprintf(`{"objectClassName":"domain","ldhName":%q}`, input)), nil
		})
		seedBootstrap(t, c, `[[["com","xn--kpry57d"],["https://rdap.registry.test/"]]]`)
		c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) { t.Fatal("unexpected WHOIS"); return nil, nil }
		if _, err := c.Lookup(context.Background(), input); err != nil {
			t.Fatal(input, err)
		}
	}
}

func TestTWSpecialNamespacesRemainDistinctFromAvailability(t *testing.T) {
	parent := "Example Education Registry\n\n Domain Name: edu.tw\n Contact:\n  Example Administrator admin@example.test\n"
	for _, domain := range []string{"example.edu.tw", "example-unregistered.edu.tw", "child.example.edu.tw"} {
		r, err := parseWHOIS(domain, parent)
		if err != nil || r.Domain != domain || r.Result != NotProvided || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
			t.Fatal(domain, r, err)
		}
	}
	for _, domain := range []string{"example.gov.tw", "example-unregistered.gov.tw"} {
		r, err := parseWHOIS(domain, "ReferralServer: rwhois://rs.gsn.gov.tw\n")
		if err != nil || r.Result != NotProvided {
			t.Fatal(domain, r, err)
		}
	}
	for _, domain := range []string{"example.mil.tw", "example-unregistered.mil.tw"} {
		r, err := parseWHOIS(domain, "網域名稱不合規定\n")
		if err != nil || r.Result != NotProvided {
			t.Fatal(domain, r, err)
		}
	}
	for _, domain := range []string{"gov.tw", "edu.tw", "mil.tw", "net.tw", "org.tw"} {
		r, err := parseWHOIS(domain, "Domain Name: "+domain+"\n\n reserved name\n")
		if err != nil || r.Result != Reserved {
			t.Fatal(domain, r, err)
		}
	}
	for _, domain := range []string{"example.tw", "example.com.tw", "example.idv.tw"} {
		if r, err := parseWHOIS(domain, "No Found\n"); err != nil || r.Result != NoRecord {
			t.Fatal(domain, r, err)
		}
		if _, err := parseWHOIS(domain, parent); err == nil {
			t.Fatal("unrelated parent accepted", domain)
		}
	}
	for _, body := range []string{"ReferralServer: rwhois://registrar.test\n", "ReferralServer: rwhois://rs.gsn.gov.tw\nAccess denied\n"} {
		if _, err := parseWHOIS("example.gov.tw", body); err == nil {
			t.Fatal("untrusted referral or service failure became a result")
		}
	}
	if _, err := parseWHOIS("example.edu.tw", parent+"Domain Name: other.edu.tw\n"); err == nil {
		t.Fatal("contradictory namespace identities accepted")
	}
}
