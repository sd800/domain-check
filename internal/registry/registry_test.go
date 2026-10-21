package registry

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func reply(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func fixtureClient(t *testing.T, handle transportFunc) *Client {
	t.Helper()
	c := NewClient(t.TempDir())
	c.Now = func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC) }
	c.HTTP = &http.Client{Transport: handle}
	c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) {
		t.Error("WHOIS must not be called for a suffix with RDAP")
		return nil, fmt.Errorf("unexpected WHOIS query")
	}
	return c
}

func seedBootstrap(t *testing.T, c *Client, services string) {
	t.Helper()
	if err := c.writeCache("dns.json", []byte(`{"services":`+services+`}`)); err != nil {
		t.Fatal(err)
	}
}

func assertServiceMetadataOnly(t *testing.T, c *Client, want []string, forbidden ...string) {
	t.Helper()
	var files []string
	err := filepath.WalkDir(c.CacheDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(c.CacheDir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, value := range forbidden {
			if strings.Contains(string(data), value) {
				t.Errorf("lookup data was persisted in %s", relative)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	sort.Strings(want)
	if strings.Join(files, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected persistent files: %v; want %v", files, want)
	}
}

func TestDomainNormalization(t *testing.T) {
	for input, want := range map[string]string{"EXAMPLE.COM.": "example.com", "例子.中国": "xn--fsqu00a.xn--fiqs8s", "example。com": "example.com", "example.co.uk": "example.co.uk"} {
		got, err := NormalizeDomain(input)
		if err != nil || got != want {
			t.Errorf("%q: got %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"com", "https://example.com", "example.com/path", "example.com:43", "example..com", "-example.com", "example_.com", "127.0.0.1", "foo.123", "example.com\r\nregistrar.com", strings.Repeat("a", 64) + ".com"} {
		if _, err := NormalizeDomain(input); err == nil {
			t.Errorf("accepted invalid input %q", input)
		}
	}
}

func TestRegistryLinksNeverFetched(t *testing.T) {
	calls := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://registry.test/rdap/domain/example.com" {
			t.Fatalf("unexpected destination %s", r.URL)
		}
		return reply(r, 200, `{"objectClassName":"domain","ldhName":"example.com","status":["active"],"links":[{"rel":"related","type":"application/rdap+json","href":"https://registrar.test/domain/example.com"}],"entities":[{"roles":["registrar"],"links":[{"href":"https://registrar.test/entity/1"}]}]}`), nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/rdap/"]]]`)
	r, err := c.Lookup(context.Background(), "example.com")
	if err != nil || r.Domain != "example.com" || calls != 1 || r.Registrant != "" {
		t.Fatalf("%+v, %v, calls %d", r, err, calls)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json"}, "example.com", "registrar.test")
}

func TestRegistryRedirectPolicy(t *testing.T) {
	for _, target := range []string{"https://registrar.test/domain/example.com", "https://registry.test/registrar/domain/example.com", "https://registry.test/rdap/domain/other.com", "https://registry.test/rdap/domain/example.com?registrar=1", "http://registry.test/rdap/domain/example.com"} {
		t.Run(target, func(t *testing.T) {
			calls := 0
			c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				response := reply(r, 302, "")
				response.Header.Set("Location", target)
				return response, nil
			})
			seedBootstrap(t, c, `[[["com"],["https://registry.test/rdap/"]]]`)
			_, err := c.Lookup(context.Background(), "example.com")
			if e, ok := err.(*QueryError); !ok || e.Code != "redirect-blocked" || calls != 1 {
				t.Fatalf("%v, calls %d", err, calls)
			}
		})
	}
	t.Run("IANA-listed alternate registry", func(t *testing.T) {
		calls := 0
		c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Host == "registry.test" {
				response := reply(r, 301, "")
				response.Header.Set("Location", "https://alternate.test/rdap/domain/example.com")
				return response, nil
			}
			return reply(r, 200, `{"objectClassName":"domain","ldhName":"example.com"}`), nil
		})
		seedBootstrap(t, c, `[[["com"],["https://registry.test/rdap/","https://alternate.test/rdap/"]]]`)
		if _, err := c.Lookup(context.Background(), "example.com"); err != nil || calls != 2 {
			t.Fatalf("%v, calls %d", err, calls)
		}
	})
}

func TestRDAPFailuresDoNotBecomeAvailabilityOrWHOIS(t *testing.T) {
	for _, status := range []int{400, 403, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c := fixtureClient(t, func(r *http.Request) (*http.Response, error) { return reply(r, status, `{}`), nil })
			seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
			r, err := c.Lookup(context.Background(), "example.com")
			if err == nil || r.Result != "" {
				t.Fatalf("%+v, %v", r, err)
			}
		})
	}
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) { return reply(r, 404, `{"errorCode":404}`), nil })
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	r, err := c.Lookup(context.Background(), "example.com")
	if err != nil || r.Result != NoRecord {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestRDAPFields(t *testing.T) {
	r, err := parseRDAP("example.com", []byte(`{
	"objectClassName":"domain", "ldhName":"EXAMPLE.COM",
	"events":[{"eventAction":"registration","eventDate":"2003-03-10T11:19:18+08:00"},{"eventAction":"last changed","eventDate":"2025-05-15T10:25:25Z"},{"eventAction":"last update of RDAP database","eventDate":"2026-10-21T00:00:00Z"}],
	"entities":[
	{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Example Registrar"]]],"entities":[{"roles":["abuse"],"vcardArray":["vcard",[["email",{},"text","abuse@registrar.test"]]]}]},
	{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Jane"],["org",{},"text",["上海溪诚信息科技有限公司",""]],["email",{},"text","admin@example.com"]]]}],
	"nameservers":[{"ldhName":"NS1.EXAMPLE.COM."},{"ldhName":"ns1.example.com"}],"secureDNS":{"delegationSigned":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Registration != "2003-03-10 03:19:18 UTC" || r.Changed != "2025-05-15 10:25:25 UTC" || r.Expiration != "" || r.Registrant != "上海溪诚信息科技有限公司" || r.Registrar != "Example Registrar" || len(r.RegistrantEmail) != 1 || len(r.Nameservers) != 1 || r.DNSSEC != "unsigned" {
		t.Fatalf("%+v", r)
	}
	for _, body := range []string{`{}`, `{"objectClassName":"entity"}`, `{"objectClassName":"domain","ldhName":"other.com"}`, `{"objectClassName":"domain","ldhName":"example.com","errorCode":403}`, `<html>Error</html>`} {
		if _, err := parseRDAP("example.com", []byte(body)); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
	r, err = parseRDAP("example.com", []byte(`{"objectClassName":"domain","ldhName":"example.com","secureDNS":{},"entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","REDACTED FOR PRIVACY"],["email",{},"text",""]]]}]}`))
	if err != nil || r.DNSSEC != "" || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestBootstrapMonthlyCache(t *testing.T) {
	calls := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		return reply(r, 200, `{"publication":"2000-01-01T00:00:00Z","services":[[["com"],["https://registry.test/"]]]}`), nil
	})
	start := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	now := start
	c.Now = func() time.Time { return now }
	if _, err := c.bootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 2, 28, 11, 59, 59, 0, time.UTC)
	if _, err := c.bootstrap(context.Background()); err != nil || calls != 1 {
		t.Fatalf("%v, calls %d", err, calls)
	}
	now = now.Add(time.Second)
	if _, err := c.bootstrap(context.Background()); err != nil || calls != 2 {
		t.Fatalf("%v, calls %d", err, calls)
	}
	// Stale cache survives a failed refresh and remains stale for the next run.
	now = now.AddDate(0, 1, 0)
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { return reply(r, 503, ""), nil })
	warned := false
	c.Warn = func(code string) { warned = code == "stale-bootstrap" }
	if _, err := c.bootstrap(context.Background()); err != nil || !warned {
		t.Fatalf("%v, warned %v", err, warned)
	}
	// A corrupt cache must be refreshed, never used as an excuse to query WHOIS.
	if err := os.WriteFile(c.cachePath("dns.json"), []byte(`{broken`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.bootstrap(context.Background()); err == nil {
		t.Fatal("used corrupt cache")
	}
}

func TestBootstrapSuffixMatchingAndHTTPS(t *testing.T) {
	b, err := decodeBootstrap([]byte(`{"services":[[["com"],["http://registry.test/","https://registry.test/"]],[["co.uk"],["https://uk.test/"]]]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := b.endpoints("example.com"); len(got) != 2 || got[0] != "https://registry.test/" {
		t.Fatalf("%v", got)
	}
	if got := b.endpoints("example.notcom"); len(got) != 0 {
		t.Fatalf("%v", got)
	}
	if got := b.endpoints("example.co.uk"); len(got) != 1 || got[0] != "https://uk.test/" {
		t.Fatalf("%v", got)
	}
}

func TestWHOISStopsAtRegistry(t *testing.T) {
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) { t.Fatal("unexpected HTTP request"); return nil, nil })
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	var destinations []string
	c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
		destinations = append(destinations, address)
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			line, _ := bufio.NewReader(server).ReadString('\n')
			if address == "whois.iana.org:43" {
				if line != "cn\r\n" {
					t.Errorf("IANA query %q", line)
				}
				fmt.Fprint(server, "domain: CN\nwhois: whois.cnnic.cn\n")
			} else if address == "whois.cnnic.cn:43" {
				if line != "shanghai.cn\r\n" {
					t.Errorf("registry query %q", line)
				}
				fmt.Fprint(server, "Domain Name: shanghai.cn\nRegistration Time: 2003-03-10 11:19:18\nRegistrar WHOIS Server: whois.registrar.test\nWhois Server: whois.registrar.test\nRegistrant: 上海溪诚信息科技有限公司\nRegistrant Contact Email: admin@example.com\nName Server: dns1.example.cn\n")
			} else {
				t.Errorf("registrar queried: %s", address)
			}
		}()
		return client, nil
	}
	r, err := c.Lookup(context.Background(), "shanghai.cn")
	if err != nil || r.Registrant != "上海溪诚信息科技有限公司" || len(r.RegistrantEmail) != 1 || len(destinations) != 2 {
		t.Fatalf("%+v, %v, %v", r, err, destinations)
	}
	_, err = c.Lookup(context.Background(), "shanghai.cn")
	if err != nil || len(destinations) != 3 || destinations[2] != "whois.cnnic.cn:43" {
		t.Fatalf("IANA discovery was not cached: %v, %v", err, destinations)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "whois/cn.json"}, "shanghai.cn", "上海溪诚信息科技有限公司", "admin@example.com", "whois.registrar.test")
}

func TestWHOISParsing(t *testing.T) {
	response := "Domain Name: shanghai.cn\nDomain Status: ok\nRegistrant: 上海溪诚信息科技有限公司\nRegistrant Email: admin@xcjjm.com\nSponsoring Registrar: 商中在线科技股份有限公司\nRegistration Time: 2003-03-10 11:19:18\nExpiration Time: 2027-05-15 11:25:25\nName Server: DNS1.EXAMPLE.CN. 192.0.2.1\nName Server: dns1.example.cn\nDNSSEC: unsigned\n"
	r, err := parseWHOIS("shanghai.cn", response)
	if err != nil || r.Registration != "2003-03-10 11:19:18" || len(r.Nameservers) != 1 || r.RegistrantEmail[0] != "admin@xcjjm.com" || r.Registrar != "商中在线科技股份有限公司" {
		t.Fatalf("%+v, %v", r, err)
	}
	for body, want := range map[string]Result{"No matching record.": NoRecord, `No match for "EXAMPLE.CN".`: NoRecord, "Domain name is reserved": Reserved, "Registration is not allowed.": NotAllowed, "Status: free": NotRegistered} {
		r, err := parseWHOIS("example.cn", body)
		if err != nil || r.Result != want {
			t.Errorf("%q: %+v, %v", body, r, err)
		}
	}
	for _, body := range []string{"", "Your connection limit exceeded", "Access denied", "Server temporarily unavailable", "Terms: unregistered domains may be reserved.", "Domain Name: other.cn"} {
		if r, err := parseWHOIS("example.cn", body); err == nil || r.Result != "" {
			t.Errorf("%q became availability: %+v, %v", body, r, err)
		}
	}
	uk := "Domain name:\n    example.co.uk\n\nRegistrar:\n    Example Ltd\n\nRelevant dates:\n    Registered on: before Aug-1996\n    Expiry date: 15-Dec-2030\n\nName servers:\n    ns1.example.co.uk 192.0.2.1\n    ns2.example.co.uk\n"
	r, err = parseWHOIS("example.co.uk", uk)
	if err != nil || r.Registrar != "Example Ltd" || len(r.Nameservers) != 2 || r.Registration != "before Aug-1996" {
		t.Fatalf("%+v, %v", r, err)
	}
}
