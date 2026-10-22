package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestNamespaceParentsDoNotDiscloseChildRecords(t *testing.T) {
	for _, pair := range [][2]string{{"institute.ac.rs", "ac.rs"}, {"ministry.gov.rs", "gov.rs"}, {"unit.mil.rs", "mil.rs"}, {"school.edu.tw", "edu.tw"}, {"office.gov.pw", "gov.pw"}} {
		domain, parent := pair[0], pair[1]
		body := `{"objectClassName":"domain","ldhName":"` + parent + `","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Parent Owner"]]]}]}`
		r, err := parseRDAP(domain, []byte(body))
		if err != nil || r.Domain != domain || r.Result != NotProvided || r.Registrant != "" {
			t.Fatalf("RDAP parent %s: %+v %v", domain, r, err)
		}
		r, err = parseWHOIS(domain, "Domain Name: "+parent+"\nRegistrant: Parent Owner\nCreation Date: 2000-01-01\n")
		if err != nil || r.Result != NotProvided || r.Registration != "" || r.Registrant != "" {
			t.Fatalf("WHOIS parent %s: %+v %v", domain, r, err)
		}
	}
	for _, pair := range [][2]string{{"child.example.com", "example.com"}, {"user.github.io", "github.io"}, {"x.gov.rs", "other.rs"}, {"x.gov.rs", "rs"}} {
		if namespaceParent(pair[0], pair[1]) {
			t.Fatal("trusted unrelated/private/non-namespace parent", pair)
		}
		if _, err := parseWHOIS(pair[0], "Domain Name: "+pair[1]+"\n"); err == nil {
			t.Fatal("accepted unrelated mismatch", pair)
		}
	}
	for _, body := range []string{"Domain Name: ac.rs\nDomain Name: other.rs\n", "Domain Name: institute.ac.rs\nDomain Name: ac.rs\n", "Domain Name: ac.rs\nNo match\n"} {
		if _, err := parseWHOIS("institute.ac.rs", body); err == nil {
			t.Fatal("accepted contradictory namespace reply", body)
		}
	}
	if _, err := parseRDAP("x.gov.pw", []byte(`{"objectClassName":"domain","ldhName":"gov.pw","unicodeName":"other.pw"}`)); err == nil {
		t.Fatal("ignored conflicting Unicode identity")
	}
}

const jiscRecord = `Domain:
 example.ac.uk
Registered For:
 Example Institution
Domain Owner:
 Example Institution
Registered By:
 Example Registry Agent
Servers:
 ns1.example.ac.uk 192.0.2.1
 ns1.example.ac.uk 2001:db8::1
 ns2.example.ac.uk 192.0.2.2
Registrant Contact:
 Domain Office
Registrant Address:
 Technology Services
 Example Street
 +44 1234 567890 (Phone)
 office@example.ac.uk
Renewal date:
 Wednesday 26th Jul 2028
Entry updated:
 Sunday 26th April 2026
Entry created:
 Wednesday 17th September 2003
`

func TestJiscNamespaceRecord(t *testing.T) {
	r, err := parseWHOIS("example.ac.uk", jiscRecord)
	if err != nil || r.Registrant != "Example Institution" || r.Registrar != "Example Registry Agent" || r.Registration != "2003-09-17" || r.Changed != "2026-04-26" || r.Expiration != "2028-07-26" || len(r.Nameservers) != 2 || strings.Join(r.RegistrantEmail, ",") != "office@example.ac.uk" {
		t.Fatalf("incomplete Jisc record: %+v %v", r, err)
	}
	redacted := strings.ReplaceAll(jiscRecord, "Example Institution", "REDACTED")
	if r, err := parseWHOIS("example.ac.uk", redacted); err != nil || r.Registrant != "" {
		t.Fatal("promoted a contact or postal line into ownership", r, err)
	}
	r, err = parseWHOIS("example-unregistered.ac.uk", "No such domain example-unregistered.ac.uk\n")
	if err != nil || r.Result != NoRecord {
		t.Fatal(r, err)
	}
	if _, err = parseWHOIS("example.ac.uk", "No such domain other.ac.uk\n"); err == nil {
		t.Fatal("accepted unrelated negative identity")
	}
}

func TestNamespaceServicesAreFixedRegistryRoutes(t *testing.T) {
	for _, domain := range []string{"example.ac.uk", "example.gov.uk"} {
		c := NewClient(t.TempDir())
		requests := 0
		c.HTTP.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.URL.String() != "https://rdap.nominet.uk/gov.uk/domain/example.gov.uk" {
				t.Fatalf("unexpected registry URL %s", req.URL)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"objectClassName":"domain","ldhName":"example.gov.uk"}`)), Header: http.Header{}}, nil
		})
		whoisCalls := 0
		c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
			whoisCalls++
			if address != "whois.ja.net:43" {
				t.Fatalf("unexpected registry %s", address)
			}
			a, b := net.Pipe()
			go func() {
				defer b.Close()
				query := make([]byte, 256)
				n, _ := b.Read(query)
				if string(query[:n]) != "example.ac.uk\r\n" {
					t.Errorf("unexpected query %q", query[:n])
				}
				io.WriteString(b, jiscRecord)
			}()
			return a, nil
		}
		r, err := c.Lookup(context.Background(), domain)
		if err != nil || r.Domain != domain {
			t.Fatal(r, err)
		}
		if domain == "example.ac.uk" && (whoisCalls != 1 || requests != 0) || domain == "example.gov.uk" && (whoisCalls != 0 || requests != 1) {
			t.Fatal("unexpected transports", domain, whoisCalls, requests)
		}
	}
	for _, domain := range []string{"ac.uk", "gov.uk", "example.co.uk", "example.gov.uk.evil.test", "example.ac.uk.evil.test"} {
		if r, w := namespaceService(domain); r != "" || w != "" {
			t.Fatal("overbroad service override", domain)
		}
	}
}

func TestDelegatedUKAbsenceIsNotChildAvailability(t *testing.T) {
	for _, domain := range []string{"hospital.nhs.uk", "force.police.uk", "unit.mod.uk", "unit.mil.uk", "office.parliament.uk"} {
		c := NewClient("")
		for _, status := range []int{404, 403, 200} {
			c.HTTP.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"objectClassName":"domain","ldhName":"` + domain + `"}`)), Header: http.Header{}}, nil
			})
			r, err := c.rdap(context.Background(), domain, []string{"https://rdap.nominet.uk/uk/"})
			if status == 404 && (err != nil || r.Result != NotProvided) || status == 403 && err == nil || status == 200 && (err != nil || r.Result != "") {
				t.Fatal(domain, status, r, err)
			}
		}
	}
	for _, domain := range []string{"nhs.uk", "example.co.uk", "hospital.nhs.uk.evil.test"} {
		if registryAbsence(domain, rdapURL("https://rdap.nominet.uk/uk/", domain)) != NoRecord {
			t.Fatal("overbroad absence rule", domain)
		}
	}
	if registryAbsence("hospital.nhs.uk", "https://other.test/domain/hospital.nhs.uk") != NoRecord {
		t.Fatal("trusted an unrelated endpoint")
	}
}

func TestNamespaceFailuresWithoutEligibleWHOISAndBlockedRedirects(t *testing.T) {
	for _, domain := range []string{"example.ac.uk", "example.gov.uk"} {
		c := NewClient(t.TempDir())
		// No country-code classification means no protocol fallback.
		seedTLDs(t, c, map[string]string{"uk": ""})
		httpCalls, whoisCalls := 0, 0
		c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) {
			whoisCalls++
			return nil, fmt.Errorf("registry unavailable")
		}
		c.HTTP.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
			httpCalls++
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("unavailable")), Header: http.Header{}}, nil
		})
		if _, err := c.Lookup(context.Background(), domain); err == nil {
			t.Fatal("service failure became a record")
		}
		if domain == "example.ac.uk" && (whoisCalls != 1 || httpCalls != 0) || domain == "example.gov.uk" && (httpCalls != 1 || whoisCalls != 0) {
			t.Fatal("failure caused a fallback", domain, httpCalls, whoisCalls)
		}
	}
	c := NewClient("")
	calls := 0
	c.HTTP.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 301, Header: http.Header{"Location": []string{"https://rdap.nominet.uk/uk/domain/example.gov.uk"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if _, err := c.Lookup(context.Background(), "example.gov.uk"); err == nil || calls != 1 {
		t.Fatal("followed a redirect outside the selected namespace route", err, calls)
	}
}

func TestIrishRestrictionAndOperationalReplies(t *testing.T) {
	body := "Error code: 01044\nError message: The domain name requested has usage restrictions applied to it. Please see your Registrar for more details.\n% Footer\n"
	r, err := parseWHOIS("mil.ie", body)
	if err != nil || r.Result != UsageRestricted {
		t.Fatal(r, err)
	}
	for _, domain := range []string{"edu.nz", "gov.sx"} {
		r, err := parseWHOIS(domain, body)
		if err != nil || r.Result != UsageRestricted {
			t.Fatal(r, err)
		}
	}
	for _, text := range []string{body + "Domain Name: other.ie\n", strings.ReplaceAll(body, "01044", "01045"), "Too many requests.\n", "Wrong second level domain name in query\n", "WARNING: query rate is now high, please try it again 1 minute(s) later.\n", "Cadena_Invalida/Invalid_String\n", "Error: subdomains such as example.ac.aw are not supported\n"} {
		if _, err := parseWHOIS("mil.ie", text); err == nil {
			t.Fatal("operational/contradictory reply became availability", text)
		}
	}
}

func TestNamespaceQuerySyntaxErrorsHaveNoAvailability(t *testing.T) {
	for _, body := range []string{"Error: subdomains such as example.ac.aw are not supported\n", "Too many requests.\n", "% Maximum query rate reached\n", "% Error: invalid query format\n"} {
		_, err := parseWHOIS("example.ac.aw", body)
		if q, ok := err.(*QueryError); !ok || q.Code != "whois-failed" {
			t.Fatal("query/service rejection was not an operational error", body, err)
		}
	}
}

func TestJPRSReservedCategoryDelegation(t *testing.T) {
	for _, domain := range []string{"ac.jp", "ed.jp", "go.jp", "lg.jp"} {
		body := "a. [Domain Name] " + strings.ToUpper(domain) + "\np. [Name Server] ns.example.jp\n[State] Reserved\n[Last Update] 2005/03/30 17:37:52 (JST)\n"
		r, err := parseWHOIS(domain, body)
		if err != nil || r.Result != Reserved || len(r.Nameservers) != 0 {
			t.Fatalf("reserved category %s: %+v %v", domain, r, err)
		}
		if _, err := parseWHOIS("example.go.jp", strings.ReplaceAll(body, strings.ToUpper(domain), "EXAMPLE.GO.JP")); err == nil {
			t.Fatal("relaxed reservation contradiction outside category name")
		}
	}
}

func TestSenegalWHOISEnvelope(t *testing.T) {
	body := `Nom de domaine: example.edu.sn
Date de création: 2001-01-02T00:00:00Z
Dernière modification: 2026-01-02T01:02:03Z
Date d'expiration: 2027-01-02T00:00:00Z
Registrar: Example Registrar
Statut: actif
[BILLING_C]
Nom: Billing Person
Courriel: billing@example.sn
[ADMIN_C]
Nom: Admin Person
Courriel: admin@example.sn
[TECH_C]
Nom: Tech Person
[HOLDER]
Nom: Example School
Courriel: owner@example.sn
Dernière modification: 2026-03-04T00:00:00Z
Serveur de noms: ns1.example.sn
DNSSEC: unsigned
`
	for _, value := range []string{body, strings.ReplaceAll(body, "\n", `\r\n`)} {
		data, _ := json.Marshal(map[string]string{"whois43response": value})
		r, err := parseRDAP("example.edu.sn", data)
		if err != nil || r.Registrant != "Example School" || r.Changed != "2026-01-02 01:02:03 UTC" || r.Registration == "" || r.Expiration == "" || strings.Join(r.RegistrantEmail, ",") != "owner@example.sn" || len(r.Nameservers) != 1 || r.DNSSEC != "unsigned" {
			t.Fatalf("envelope: %+v %v", r, err)
		}
		if _, err := parseRDAP("example.edu.test", data); err == nil {
			t.Fatal("accepted envelope outside .sn")
		}
	}
	for _, data := range []string{`{"whois43response":"%% NOT FOUND"}`, `{"whois43response":"%% NOT FOUND\\r\\n"}`} {
		r, err := parseRDAP("example.edu.sn", []byte(data))
		if err != nil || r.Result != NoRecord {
			t.Fatal(r, err)
		}
	}
	for _, data := range []string{`{"whois43response":"Domain: other.sn"}`, `{"whois43response":42}`, `{"whois43response":"No match","extra":true}`, `{"errorCode":503,"whois43response":"No match"}`} {
		if _, err := parseRDAP("example.edu.sn", []byte(data)); err == nil {
			t.Fatal("accepted invalid/contradictory envelope", data)
		}
	}
}
