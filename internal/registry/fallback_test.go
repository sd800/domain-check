package registry

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

func seedFallbackTLD(t *testing.T, c *Client, tld, kind, status, host string) {
	t.Helper()
	var server any
	if host != "" {
		server = host
	}
	data, _ := json.Marshal(map[string]any{"meta": map[string]int{"specversion": 1}, "domains": []map[string]any{{"name": tld, "type": kind, "status": status, "whois_server": server}}})
	if err := c.writeCache("tlds.json", data); err != nil {
		t.Fatal(err)
	}
}

func fallbackWHOIS(t *testing.T, c *Client, domain, host, body string) *int {
	t.Helper()
	calls := new(int)
	c.DialWHOIS = func(_ context.Context, _, address string) (net.Conn, error) {
		*calls++
		if address != host+":43" {
			t.Errorf("unexpected fallback destination %q", address)
		}
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			query, _ := bufio.NewReader(b).ReadString('\n')
			if query != registryWHOISQuery(host, domain)+"\r\n" {
				t.Errorf("unexpected fallback query %q", query)
			}
			fmt.Fprint(b, body)
		}()
		return a, nil
	}
	return calls
}

func TestCountryCodeRDAPFailureUsesOnlyPublishedRegistryWHOIS(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		transport bool
	}{{"server", 503, "", false}, {"access", 403, "", false}, {"tls/network", 0, "", true}, {"bad JSON", 200, "{broken", false}, {"wrong identity", 200, `{"objectClassName":"domain","ldhName":"other.zz"}`, false}} {
		t.Run(tc.name, func(t *testing.T) {
			c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "rdap.registry.test" {
					t.Fatalf("unexpected request %s", req.URL)
				}
				if tc.transport {
					return nil, errors.New("registry transport unavailable")
				}
				return reply(req, tc.status, tc.body), nil
			})
			seedBootstrap(t, c, `[[["zz"],["https://rdap.registry.test/"]]]`)
			seedFallbackTLD(t, c, "zz", "country-code", "active", "whois.registry.test")
			calls := fallbackWHOIS(t, c, "example.zz", "whois.registry.test", "Domain Name: example.zz\nRegistrant: Example Owner\nCreation Date: 2000-01-02\nRegistrar WHOIS Server: whois.registrar.test\nReferralServer: whois://registrar.test\n")
			r, err := c.Lookup(context.Background(), "example.zz")
			if err != nil || r.Registrant != "Example Owner" || r.Registration != "2000-01-02" || *calls != 1 {
				t.Fatal(r, err, *calls)
			}
			assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, "example.zz", "Example Owner", "registrar.test")
		})
	}
}

func TestFallbackRequiresActivePublishedRegistryService(t *testing.T) {
	for _, tc := range []struct{ tld, kind, status, host string }{{"zz", "generic", "retired", "whois.registry.test"}, {"zz", "country-code", "retired", "whois.registry.test"}, {"zz", "generic", "active", ""}, {"zz", "country-code", "active", ""}} {
		c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
		seedBootstrap(t, c, fmt.Sprintf(`[[[%q],["https://rdap.registry.test/"]]]`, tc.tld))
		seedFallbackTLD(t, c, tc.tld, tc.kind, tc.status, tc.host)
		_, err := c.Lookup(context.Background(), "example."+tc.tld)
		if q, ok := err.(*QueryError); !ok || q.Code != "rdap-failed" {
			t.Fatal(tc, err)
		}
	}
	for _, service := range []struct{ tld, kind string }{{"com", "generic"}, {"test", "generic"}, {"zz", ""}, {"xn--p1ai", "country-code"}, {"xn--p1ai", "generic"}} {
		c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
		seedBootstrap(t, c, fmt.Sprintf(`[[[%q],["https://rdap.registry.test/"]]]`, service.tld))
		seedFallbackTLD(t, c, service.tld, service.kind, "active", "whois.registry.test")
		domain := "example." + service.tld
		calls := fallbackWHOIS(t, c, domain, "whois.registry.test", "Domain Name: "+domain+"\n")
		if _, err := c.Lookup(context.Background(), domain); err != nil || *calls != 1 {
			t.Fatal(err, *calls)
		}
	}
}

func TestSuccessfulAndLimitedRDAPRepliesDoNotFallback(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{{200, `{"objectClassName":"domain","ldhName":"example.zz"}`}, {404, ""}, {429, ""}, {302, ""}} {
		c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "rdap.registry.test" {
				t.Fatalf("unexpected discovery on a completed/stopped lookup: %s", req.URL)
			}
			r := reply(req, tc.status, tc.body)
			if tc.status == 302 {
				r.Header.Set("Location", "https://registrar.test/domain/example.zz")
			}
			return r, nil
		})
		seedBootstrap(t, c, `[[["zz"],["https://rdap.registry.test/"]]]`)
		// No TLD cache is needed on success, no-record, rate-limit or blocked redirect.
		r, err := c.Lookup(context.Background(), "example.zz")
		if tc.status == 200 && (err != nil || r.Result != "") || tc.status == 404 && (err != nil || r.Result != NoRecord) || tc.status >= 400 && tc.status != 404 && err == nil || tc.status == 302 && err == nil {
			t.Fatal(tc, r, err)
		}
		if _, err := os.Stat(c.cachePath("tlds.json")); !os.IsNotExist(err) {
			t.Fatal("unneeded WHOIS discovery was retained", err)
		}
	}
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return nil, context.Canceled })
	seedBootstrap(t, c, `[[["zz"],["https://rdap.registry.test/"]]]`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Lookup(ctx, "example.zz"); err == nil {
		t.Fatal("cancellation became success")
	}
}

func TestFallbackPreservesRestrictionsAndBothFailures(t *testing.T) {
	for _, tc := range []struct {
		body string
		want Result
	}{{"No match for example.zz\n", NoRecord}, {"Domain Name: example.zz\nStatus: reserved\n", Reserved}, {"Domain Name: gov.pw\nRegistrant: Parent Owner\n", NotProvided}} {
		domain, tld := "example.zz", "zz"
		if tc.want == NotProvided {
			domain, tld = "office.gov.pw", "pw"
		}
		c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
		seedBootstrap(t, c, fmt.Sprintf(`[[[%q],["https://rdap.registry.test/"]]]`, tld))
		seedFallbackTLD(t, c, tld, "country-code", "active", "whois.registry.test")
		fallbackWHOIS(t, c, domain, "whois.registry.test", tc.body)
		r, err := c.Lookup(context.Background(), domain)
		if err != nil || !reflect.DeepEqual(r, Record{Domain: domain, Result: tc.want}) {
			t.Fatal(r, err)
		}
	}
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
	seedBootstrap(t, c, `[[["zz"],["https://rdap.registry.test/"]]]`)
	seedFallbackTLD(t, c, "zz", "country-code", "active", "whois.registry.test")
	c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("unavailable\x1b[31m") }
	_, err := c.Lookup(context.Background(), "example.zz")
	if q, ok := err.(*QueryError); !ok || q.Code != "fallback-failed" || !strings.Contains(q.Detail, "HTTP 503") || !strings.Contains(q.Detail, "WHOIS:") || strings.Contains(q.Detail, "\x1b") {
		t.Fatal(err)
	}
}

func TestFallbackCannotUseInvalidMetadataOrRDAPReferrals(t *testing.T) {
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "rdap.registry.test" {
			return reply(req, 503, ""), nil
		}
		if req.URL.String() != "https://data.iana.org/TLD/v1/tlds.json" {
			t.Fatal(req.URL)
		}
		return reply(req, 200, `{"meta":{"specversion":1},"domains":[{"name":"zz","type":"country-code","status":"active","whois_server":"https://registrar.test"}]}`), nil
	})
	seedBootstrap(t, c, `[[["zz"],["https://rdap.registry.test/"]]]`)
	if _, err := c.Lookup(context.Background(), "example.zz"); err == nil {
		t.Fatal("invalid fallback metadata was accepted")
	}
}

func TestNominetPublishedWHOISOverrideAndDelegatedNamespaces(t *testing.T) {
	for _, domain := range []string{"example.uk", "office.gov.uk", "unit.mil.uk"} {
		c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 503, ""), nil })
		seedBootstrap(t, c, `[[["uk"],["https://rdap.registry.test/"]]]`)
		seedFallbackTLD(t, c, "uk", "country-code", "active", "")
		body := "Domain name: " + domain + "\nRegistrant: Example Institution\n"
		want := Result("")
		if domain != "example.uk" {
			parent := domain[strings.IndexByte(domain, '.')+1:]
			body = "Domain name: " + parent + "\nRegistrant: Parent Institution\n"
			want = NotProvided
		}
		calls := fallbackWHOIS(t, c, domain, "whois.nic.uk", body)
		r, err := c.Lookup(context.Background(), domain)
		if err != nil || r.Result != want || *calls != 1 || want == NotProvided && r.Registrant != "" {
			t.Fatal(r, err, *calls)
		}
	}
	if (tldService{CountryCode: false}).whoisHost("uk") != "" || (tldService{CountryCode: true}).whoisHost("na") != "" {
		t.Fatal("guessed an unpublished service")
	}
}

func TestBlockedRedirectStopsAllProtocolAttempts(t *testing.T) {
	calls := 0
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Host != "rdap.registry.test" {
			t.Fatalf("request after blocked redirect: %s", req.URL)
		}
		r := reply(req, 302, "")
		r.Header.Set("Location", "https://registrar.test/domain/example.zz")
		return r, nil
	})
	seedBootstrap(t, c, `[[["zz"],["https://rdap.registry.test/","https://alternate.registry.test/"]]]`)
	seedFallbackTLD(t, c, "zz", "country-code", "active", "whois.registry.test")
	_, err := c.Lookup(context.Background(), "example.zz")
	if q, ok := err.(*QueryError); !ok || q.Code != "redirect-blocked" || calls != 1 {
		t.Fatal(err, calls)
	}
}
