package registry

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestIDNWHOISUsesBulkIANARegistry(t *testing.T) {
	for _, suffix := range []string{"中国", "中國", "香港", "台湾", "澳門"} {
		t.Run(suffix, func(t *testing.T) {
			domain := idnName(t, "示例."+suffix)
			tld := domain[strings.LastIndexByte(domain, '.')+1:]
			c := fixtureClient(t, func(*http.Request) (*http.Response, error) {
				t.Fatal("unexpected HTTP request")
				return nil, nil
			})
			seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
			seedTLDs(t, c, map[string]string{tld: "whois.registry.test"})
			calls := 0
			c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
				calls++
				if address != "whois.registry.test:43" {
					t.Errorf("queried IANA discovery or registrar: %s", address)
				}
				client, server := net.Pipe()
				go func() {
					defer server.Close()
					query, _ := bufio.NewReader(server).ReadString('\n')
					if query != domain+"\r\n" {
						t.Errorf("unexpected domain query: %q", query)
					}
					fmt.Fprintf(server, "Domain Name: %s\nRegistrant: 示例机构\nRegistrar WHOIS Server: whois.registrar.test\n", domain)
				}()
				return client, nil
			}
			for iteration := 0; iteration < 2; iteration++ {
				r, err := c.Lookup(context.Background(), domain)
				if err != nil || r.Registrant != "示例机构" || calls != 1+iteration {
					t.Fatalf("%+v, %v, calls=%d", r, err, calls)
				}
			}
			assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, domain, "示例机构", "whois.registrar.test")
		})
	}
}

func TestIDNWHOISDiscoveryRejectsOtherSuffixes(t *testing.T) {
	c := fixtureClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unnecessary network request")
		return nil, nil
	})
	seedTLDs(t, c, map[string]string{"xn--fiqz9s": "whois.registry.test"})
	if host, err := c.whoisServer(context.Background(), "xn--fiqs8s"); err == nil || host != "" {
		t.Fatalf("a different IDN suffix was accepted: %q, %v", host, err)
	}
}

func idnName(t *testing.T, value string) string {
	t.Helper()
	name, err := NormalizeDomain(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func TestCNNICDeclaredIDNBundle(t *testing.T) {
	cn, traditional := idnName(t, "示例.中国"), idnName(t, "示例.中國")
	body := fmt.Sprintf("Domain Name: 示例.中国\nPuny Name: %s\nDomain Name: 示例.中國\nPuny Name: %s\nROID: SYNTHETIC\nRegistrant: 示例机构\nRegistrant Email: owner@example.test\nRegistration Time: 2000-10-21 01:02:03\nExpiration Time: 2030-10-21 01:02:03\nName Server: ns1.example.test\n", cn, traditional)
	for _, domain := range []string{cn, traditional} {
		r, err := parseWHOIS(domain, body)
		if err != nil || r.Registrant != "示例机构" || r.Registration != "2000-10-21 01:02:03" || r.Expiration != "2030-10-21 01:02:03" || len(r.Nameservers) != 1 || len(r.RegistrantEmail) != 1 {
			t.Fatalf("%+v, %v", r, err)
		}
	}
	if _, err := parseWHOIS(cn, strings.Replace(body, "Puny Name: "+cn, "Puny Name: other.cn", 1)); err == nil {
		t.Fatal("accepted contradictory Unicode and ASCII names")
	}
	if _, err := parseWHOIS(idnName(t, "其他.中国"), body); err == nil {
		t.Fatal("bundle became a record for an unrelated name")
	}
}

func TestHKCompactIDNBundle(t *testing.T) {
	domain := idnName(t, "示例.香港")
	for _, value := range []string{"示例.HK /.香港  Bundled Domain Name: EXAMPLE.HK", "示例.HK/.香港 bundled domain name: EXAMPLE.HK"} {
		r, err := parseWHOIS(domain, "Domain Name: "+value+`
Domain Name Commencement Date: 09-11-2000
Expiry Date: 07-06-2030
DNSSEC: unsigned / unsigned
Registrant Contact Information:
Company English Name: Example University
Email: owner@example.test
Name Servers Information:
ns1.example.test
`)
		if err != nil || r.Registration != "2000-11-09" || r.Expiration != "2030-06-07" || r.Registrant != "Example University" || len(r.RegistrantEmail) != 1 || r.DNSSEC != "unsigned" || len(r.Nameservers) != 1 {
			t.Fatalf("%s: %+v, %v", value, r, err)
		}
	}
	for _, value := range []string{"other.hk/.香港", "示例.hk unrelated.hk", "示例.hk/.invalid:443"} {
		if _, err := parseWHOIS(domain, "Domain Name: "+value); err == nil {
			t.Fatalf("unrelated or malformed alias accepted: %s", value)
		}
	}
	if got := normalizeBundledDNSSEC("signed / unsigned"); got != "signed / unsigned" {
		t.Fatal("mixed DNSSEC values were guessed")
	}
}

func TestTWNICDeclaredCharacterVariants(t *testing.T) {
	primary, simplified := idnName(t, "台灣示例.台灣"), idnName(t, "台湾示例.台湾")
	body := fmt.Sprintf(`註冊原型域名: 台灣示例.台灣 (%s)
繁體對照字域名: 台灣示例.台灣 (%s)
簡体對照字域名: 台湾示例.台湾 (%s)
保留字域名(由下列相關字組合之域名):
 台 臺
 湾 灣
 示
 例
 Domain Status: inactive
 Registrant:
  示例大學
  Example University
  owner@example.test
  TW
 Administrative Contact:
  (Redacted for privacy)
 Technical Contact:
  Wrong Technical Contact
  tech@example.test
 Record expires on  (UTC+8)
 Record created on 2000-10-21 20:54:19 (UTC+8)
 Domain servers in listed order:
  ns1.example.test
Registration Service Provider: Example Registrar
 (reserved word)
`, primary, primary, simplified)
	for _, value := range []string{"台湾示例.台湾", "臺灣示例.台湾"} {
		r, err := parseWHOIS(idnName(t, value), body)
		if err != nil || r.Result != "" || r.Registrant != "示例大學" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" || r.Registration != "2000-10-21 12:54:19 UTC" || r.Expiration != "" || r.Registrar != "Example Registrar" || len(r.Nameservers) != 1 {
			t.Fatalf("%s: %+v, %v", value, r, err)
		}
	}
	for _, value := range []string{"其他示例.台湾", "臺灣示例.中国", "child.台湾示例.台湾"} {
		if _, err := parseWHOIS(idnName(t, value), body); err == nil {
			t.Fatalf("undeclared variant accepted: %s", value)
		}
	}
	if _, err := parseWHOIS(simplified, strings.Replace(body, "台 臺", "台", 1)); err != nil {
		t.Fatal("an explicitly declared alias was lost")
	}
	if _, err := parseWHOIS(idnName(t, "臺灣示例.台湾"), strings.Replace(body, "台 臺", "台", 1)); err == nil {
		t.Fatal("character not in the variant table was accepted")
	}
}

func TestIDNNegativeRepliesAndMissingFields(t *testing.T) {
	domain := idnName(t, "示例.澳門")
	for _, body := range []string{"No match for 示例.澳門\n", "No Found\n"} {
		r, err := parseWHOIS(domain, body)
		if err != nil || r.Result != NoRecord {
			t.Fatalf("%+v, %v", r, err)
		}
	}
	if _, err := parseWHOIS(domain, "No match for 其他.澳門"); err == nil {
		t.Fatal("negative reply for another Unicode name was accepted")
	}
	r, err := parseWHOIS(domain, "Domain Name: 示例.澳門\nDomain Name(Punycode): "+domain+"\nRecord created on 2000-10-21 17:25:09.420154\nRecord expires on 2030-10-21\n")
	if err != nil || r.Registration != "2000-10-21 17:25:09" || r.Expiration != "2030-10-21" || r.Registrant != "" || r.Registrar != "" || len(r.Nameservers) != 0 {
		t.Fatalf("%+v, %v", r, err)
	}
	r, err = parseWHOIS(idnName(t, "示例.香港"), "This domain is currently not available for registration. Please select other domain.")
	if err != nil || r.Result != Unavailable {
		t.Fatalf("%+v, %v", r, err)
	}
	if r, err := parseWHOIS(idnName(t, "示例.香港"), "CMM174"); err == nil || r.Result != "" {
		t.Fatal("unrecognized registry code became availability")
	}
}

func TestInternationalizedSuffixesPreferListedRDAP(t *testing.T) {
	for _, value := range []string{"example.vermögensberater", "example.vermögensberatung", "example.みんな", "example.コム", "example.한국", "example.삼성", "example.台灣"} {
		t.Run(value, func(t *testing.T) {
			domain := idnName(t, value)
			tld := domain[strings.LastIndexByte(domain, '.')+1:]
			calls := 0
			c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.String() != "https://registry.test/domain/"+domain {
					t.Fatalf("unexpected endpoint: %s", req.URL)
				}
				return reply(req, 200, fmt.Sprintf(`{"objectClassName":"domain","ldhName":%q}`, domain)), nil
			})
			seedBootstrap(t, c, fmt.Sprintf(`[[[%q],["https://registry.test/"]]]`, tld))
			c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) {
				t.Fatal("an IANA-listed RDAP suffix used WHOIS")
				return nil, nil
			}
			r, err := c.Lookup(context.Background(), domain)
			if err != nil || r.Domain != domain || calls != 1 {
				t.Fatalf("%+v, %v, calls=%d", r, err, calls)
			}
		})
	}
}

func TestWHOISWrappedRedactionIsHidden(t *testing.T) {
	for _, placeholder := range []string{"[REDACTED FOR PRIVACY]", "(Redacted for privacy)", "[Redacted]"} {
		r, err := parseWHOIS("example.test", "Domain Name: example.test\nRegistrant: "+placeholder+"\nRegistrant Email: "+placeholder+"\n")
		if err != nil || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
			t.Fatalf("%+v, %v", r, err)
		}
	}
}
