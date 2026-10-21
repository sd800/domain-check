package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/idna"
)

func TestEmojiDomainNormalizationAndDisplay(t *testing.T) {
	for _, test := range []struct{ input, ascii, display string }{
		{"😀.ws", "xn--e28h.ws", "😀.ws"},
		{"☕.ws", "xn--53h.ws", "☕.ws"},
		{"❤️.ws", "xn--qei.ws", "❤.ws"},
		{"👩‍💻.ws", "xn--1ug1855pcha.ws", "👩‍💻.ws"},
		{"🇨🇳.ws", "xn--g77hwa.ws", "🇨🇳.ws"},
	} {
		for _, input := range []string{test.input, test.ascii, strings.ToUpper(test.ascii) + "."} {
			got, err := NormalizeDomain(input)
			if err != nil || got != test.ascii || UnicodeDomain(got) != test.display {
				t.Fatalf("%s: %s, %v, display=%q", input, got, err, UnicodeDomain(got))
			}
		}
		for _, input := range []string{"https://www." + test.input + "/?a=1&b=2", "https://user:secret@www." + test.ascii + ":8443/path#fragment"} {
			if got, err := NormalizeInput(input); err != nil || got != test.ascii {
				t.Fatalf("emoji URL: %s, %v; want %s", got, err, test.ascii)
			}
		}
	}
	for _, name := range []string{"👩🏽‍💻.ws", "👨‍👩‍👧‍👦.ws", "🏳️‍🌈.ws", "1️⃣.ws"} {
		ascii, err := NormalizeDomain(name)
		if err != nil || UnicodeDomain(ascii) == "" {
			t.Fatalf("emoji sequence rejected: %s, %s, %v", name, ascii, err)
		}
		if roundTrip, err := NormalizeDomain(UnicodeDomain(ascii)); err != nil || roundTrip != ascii {
			t.Fatalf("emoji sequence identity changed: %s, %v", roundTrip, err)
		}
	}
}

func TestEmojiCompatibilityDoesNotAdmitInvalidJoinersOrControls(t *testing.T) {
	for _, label := range []string{"a‍b", "😀‍a", "a‍😀", "‍😀", "😀‍", "😀‍‍💻", "😀‍́", "😀‌💻", "😀\u202e💻", "😀\x1b", "😀/other"} {
		puny, err := idna.Punycode.ToASCII(label + ".ws")
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{label + ".ws", puny} {
			if got, err := NormalizeDomain(name); err == nil {
				t.Errorf("invalid emoji label accepted: %q as %s", name, got)
			}
			if got := UnicodeDomain(name); got != "" {
				t.Errorf("invalid display name: %q", got)
			}
		}
	}
}

func TestEmojiRegistryIdentityAndRouting(t *testing.T) {
	const domain = "xn--1ug1855pcha.ws"
	for _, name := range []string{domain, "👩‍💻.ws"} {
		r, err := parseRDAP(domain, []byte(fmt.Sprintf(`{"objectClassName":"domain","unicodeName":%q}`, name)))
		if err != nil || r.Domain != domain {
			t.Fatalf("emoji RDAP identity: %+v, %v", r, err)
		}
		for _, body := range []string{"Domain Name: " + name + "\n", "Domain Name:\n  " + name + "\n"} {
			r, err = parseWHOIS(domain, body+"Registrar WHOIS Server: whois.registrar.test\n")
			if err != nil || r.Domain != domain {
				t.Fatalf("emoji WHOIS identity: %+v, %v", r, err)
			}
		}
	}
	if _, err := parseWHOIS(domain, "Domain Name: 😀.ws\n"); err == nil {
		t.Fatal("different emoji became the queried domain")
	}
	if _, err := parseRDAP(domain, []byte(`{"objectClassName":"domain","unicodeName":"😀.ws"}`)); err == nil {
		t.Fatal("different emoji RDAP identity accepted")
	}
	calls := 0
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.String() != "https://registry.test/domain/"+domain {
			t.Fatalf("emoji query left registry endpoint: %s", req.URL)
		}
		return reply(req, 200, `{"objectClassName":"domain","unicodeName":"👩‍💻.ws"}`), nil
	})
	seedBootstrap(t, c, `[[["ws"],["https://registry.test/"]]]`)
	if r, err := c.Lookup(context.Background(), "👩‍💻.ws"); err != nil || r.Domain != domain || calls != 1 {
		t.Fatalf("emoji lookup: %+v, %v, calls=%d", r, err, calls)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json"}, domain, "👩‍💻.ws")
}
