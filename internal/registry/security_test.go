package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInjectionInputsNeverReachTransport(t *testing.T) {
	c := fixtureClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("malicious input reached HTTP")
		return nil, nil
	})
	for _, input := range []string{"example.com\r\nsecond.test", "example.com\x00", "-h.example", "example.com;id", "$(id).com", "`id`.com", "example.com/../../etc/passwd"} {
		if _, err := c.Lookup(context.Background(), input); err == nil {
			t.Errorf("accepted injection: %q", input)
		}
	}
	c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("malicious query reached WHOIS")
		return nil, nil
	}
	for _, query := range []string{"example.com\r\nsecond.test", "example.com\x00", "-h example.com", "example.com\tsecond.test", "example.com/e", "example.com "} {
		if _, err := c.whois(context.Background(), "whois.registry.test", query); err == nil {
			t.Errorf("accepted WHOIS injection: %q", query)
		}
	}
}

func TestPastedURLSpecialCharactersRemainLocal(t *testing.T) {
	input := "https://user:password@www.example.com:8443/path/$(id);echo?x=`id`&q=secret#fragment"
	domain, err := NormalizeInput(input)
	if err != nil || domain != "example.com" {
		t.Fatalf("URL normalization: %q, %v", domain, err)
	}
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://registry.test/domain/example.com" || r.Header.Get("Authorization") != "" {
			t.Fatalf("URL secrets or commands escaped local normalization: %s", r.URL)
		}
		return reply(r, 200, `{"objectClassName":"domain","ldhName":"example.com"}`), nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	if _, err := c.Lookup(context.Background(), domain); err != nil {
		t.Fatal(err)
	}
}

func TestUnsafeServiceMetadataIsRejected(t *testing.T) {
	for _, endpoint := range []string{
		"https://127.0.0.1/", "http://169.254.169.254/", "https://[::1]/", "https://registry.test:22/",
		"https://user@registry.test/", "https://registry.test/?", "https://registry.test/#", "https://registry.test/a/../", "https://registry.test/%2e%2e/",
		"https://registry.test/a%2fb/", "https://registry.test/a\\b/", "file:///etc/passwd",
	} {
		data := fmt.Sprintf(`{"services":[[["com"],[%q]]]}`, endpoint)
		if _, err := decodeBootstrap([]byte(data)); err == nil {
			t.Errorf("accepted unsafe metadata: %s", endpoint)
		}
	}
	for _, suffixes := range []string{`"com","com"`, `"../com"`, `"com\r\n"`, `""`} {
		data := `{"services":[[[` + suffixes + `],["https://registry.test/"]]]}`
		if _, err := decodeBootstrap([]byte(data)); err == nil {
			t.Errorf("accepted ambiguous suffix routing: %s", data)
		}
	}
	for _, host := range []string{"127.0.0.1", "192.168.0.1", "registry.test:43", "whois.-bad.test", "whois..test", "whois.test\r\n", strings.Repeat("a", 64) + ".test"} {
		if validWHOISHost(host) {
			t.Errorf("accepted WHOIS host: %q", host)
		}
	}
}

func TestPublicDialPinsValidatedAddresses(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "224.0.0.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::7f00:1", "2002:7f00:1::"} {
		calls := 0
		_, err := dialPublicAddresses(context.Background(), "tcp", "443", []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr(raw)}, func(context.Context, string, string) (net.Conn, error) {
			calls++
			return nil, nil
		})
		if err == nil || calls != 0 {
			t.Errorf("private or mixed DNS answer reached a socket: %s, %v", raw, err)
		}
	}
	var address string
	_, err := dialPublicAddresses(context.Background(), "tcp", "43", []netip.Addr{netip.MustParseAddr("8.8.8.8")}, func(_ context.Context, _, destination string) (net.Conn, error) {
		address = destination
		return nil, nil
	})
	if err != nil || address != "8.8.8.8:43" {
		t.Fatalf("did not dial validated IP: %q, %v", address, err)
	}
}

func TestHostileResponsesStayWithinParsingBudgets(t *testing.T) {
	for _, data := range []string{
		strings.Repeat("[", 65) + strings.Repeat("]", 65),
		`{"value":"` + strings.Repeat("x", 65537) + `"}`,
		strings.Repeat(" ", maxResponse+1),
	} {
		var value any
		if err := unmarshalResponse([]byte(data), &value); err == nil {
			t.Fatal("accepted response beyond parsing limits")
		}
	}
	var value any
	if err := unmarshalResponse([]byte(`{"text":"[[[\\\"{{{"}`), &value); err != nil {
		t.Fatalf("brackets inside strings counted as nesting: %v", err)
	}
	for _, body := range []string{strings.Repeat("\n", 4097), strings.Repeat("x", 65537)} {
		if _, err := parseWHOIS("example.com", body); err == nil {
			t.Fatal("accepted oversized WHOIS structure")
		}
	}
	statuses, _ := json.Marshal(strings.Split(strings.Repeat("ok,", 256)+"ok", ","))
	if _, err := parseRDAP("example.com", []byte(`{"objectClassName":"domain","ldhName":"example.com","status":`+string(statuses)+`}`)); err == nil {
		t.Fatal("accepted excessive RDAP field count")
	}
}

func TestRDAPCannotDisguiseAnotherUnicodeIdentity(t *testing.T) {
	if _, err := parseRDAP("example.com", []byte(`{"objectClassName":"domain","ldhName":"example.com","unicodeName":"other.com"}`)); err == nil {
		t.Fatal("accepted contradictory RDAP identities")
	}
}

func TestDNSFallbackDoesNotExecutePATHProgram(t *testing.T) {
	if _, err := os.Stat("/usr/bin/dig"); err != nil {
		t.Skip("macOS system dig is unavailable")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "executed")
	if err := os.WriteFile(filepath.Join(dir, "dig"), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := NewClient("").RunDig(ctx, "-v"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("a PATH-supplied program was executed")
	}
}

func TestExcessiveWHOISFieldsFailWithoutSilentTruncation(t *testing.T) {
	var response strings.Builder
	response.WriteString("Domain Name: example.test\nName Server:")
	for i := range 3000 {
		fmt.Fprintf(&response, " ns%d.example.test", i)
	}
	response.WriteByte('\n')
	if _, err := parseWHOIS("example.test", response.String()); err == nil {
		t.Fatal("accepted an excessive field set")
	}
}

func FuzzDomainCannotInjectWHOISOrURL(f *testing.F) {
	for _, input := range []string{"example.com", ".github.io", ".com", "cn", ".中国", "auto", "https://www.example.com/private?token=1", "😀.ws", "example.com\r\nother.com", "-h.example", "例子.中国", "a;b.com"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		domain, err := NormalizeInput(input)
		if err != nil {
			return
		}
		if strings.ContainsAny(domain, "\r\n\x00/?#@; ") {
			t.Fatalf("unsafe normalized domain: %q", domain)
		}
		if !strings.Contains(domain, ".") {
			if normalized, err := normalizeTLD(domain); err != nil || normalized != domain || validWHOISQuery("whois.registry.test", domain) {
				t.Fatalf("unsafe normalized TLD: %q", domain)
			}
			return
		}
		if !validWHOISQuery("whois.registry.test", domain) {
			t.Fatalf("unsafe normalized WHOIS domain: %q", domain)
		}
		for _, host := range []string{"whois.jprs.jp", "whois.denic.de", "whois.punktum.dk"} {
			if !validWHOISQuery(host, registryWHOISQuery(host, domain)) {
				t.Fatalf("invalid registry selector for %q", domain)
			}
		}
	})
}

func FuzzUntrustedRegistryResponses(f *testing.F) {
	for _, body := range []string{
		"Domain Name: example.test\nRegistrant: Synthetic Owner\n",
		`{"objectClassName":"domain","ldhName":"example.test"}`,
		allocationFixture,
		govAddressFixture,
		detailedWHOISFixture,
		detailedRDAPFixture,
		eduCAUSERecord,
		twRecord,
		koreanFixture,
		kyrgyzFixture,
		"Domain Name: example.test\nRegistrant: \x1b]52;c;YXR0YWNr\a\n",
		"No match for example.test",
	} {
		f.Add(body)
	}
	f.Fuzz(func(t *testing.T, body string) {
		if len(body) > 65536 {
			return
		}
		_, _ = parseWHOIS("example.test", body)
		_, _ = parseWHOIS("example.us", body)
		_, _ = parseWHOIS("example.edu", body)
		_, _ = parseWHOIS("example.com.tw", body)
		_, _ = parseWHOIS("example.ac.uk", body)
		_, _ = parseWHOIS("example.go.kr", body)
		_, _ = parseWHOIS("example.edu.kg", body)
		_, _ = parseRDAP("example.edu.sn", []byte(body))
		_, _ = parseRDAP("example.test", []byte(body))
		_, _ = parseRDAP("example.us", []byte(body))
		_, _ = parseRDAP("example.gov", []byte(body))
		_, _ = parseRDAP("zz", []byte(body))
	})
}
