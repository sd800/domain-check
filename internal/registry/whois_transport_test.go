package registry

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestWHOISRegistryQueriesAndLegacyEncodings(t *testing.T) {
	for _, test := range []struct {
		host, domain, query, owner, encoding string
	}{
		{"whois.jprs.jp", "example.jp", "example.jp/e", "Example Owner", "utf8"},
		{"whois.denic.de", "example.de", "-T dn,ace example.de", "Example Owner", "utf8"},
		{"whois.punktum.dk", "example.dk", "--charset=utf-8 example.dk", "Example Owner", "utf8"},
		{"whois.nic.hu", "example.hu", "example.hu", "Példa Owner", "latin1"},
		{"whois.dns.lu", "example.lu", "example.lu", "Exemple Société", "latin1"},
		{"whois.cnnic.cn", "example.cn", "example.cn", "示例机构", "gb18030"},
	} {
		t.Run(test.host, func(t *testing.T) {
			c := NewClient("")
			calls := 0
			c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
				calls++
				if address != test.host+":43" {
					t.Errorf("unexpected server %s", address)
				}
				client, server := net.Pipe()
				go func() {
					defer server.Close()
					line, _ := bufio.NewReader(server).ReadString('\n')
					if line != test.query+"\r\n" {
						t.Errorf("query %q; want %q", line, test.query)
					}
					data := []byte(fmt.Sprintf("Domain Name: %s\nRegistrant: %s\nRegistrar WHOIS Server: whois.registrar.test\n", test.domain, test.owner))
					var err error
					switch test.encoding {
					case "latin1":
						data, err = charmap.ISO8859_1.NewEncoder().Bytes(data)
					case "gb18030":
						data, err = simplifiedchinese.GB18030.NewEncoder().Bytes(data)
					}
					if err != nil {
						t.Error(err)
						return
					}
					_, _ = server.Write(data)
				}()
				return client, nil
			}
			body, err := c.whois(context.Background(), test.host, registryWHOISQuery(test.host, test.domain))
			if err != nil {
				t.Fatal(err)
			}
			r, err := parseWHOIS(test.domain, body)
			if err != nil || r.Registrant != test.owner || calls != 1 || strings.Contains(r.Registrant, "�") {
				t.Fatalf("%+v, %v, calls=%d", r, err, calls)
			}
		})
	}
}

func TestWHOISIncludedObjectsDoNotCauseAdditionalQueries(t *testing.T) {
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		t.Fatal("WHOIS object URL was fetched")
		return nil, nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	seedTLDs(t, c, map[string]string{"at": "whois.nic.at"})
	var destinations []string
	c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
		destinations = append(destinations, address)
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			_, _ = bufio.NewReader(server).ReadString('\n')
			switch address {
			case "whois.nic.at:43":
				fmt.Fprint(server, "domain: example.at\nregistrant: SHARED1\nnsset: SHARED1\nkeyset: SHARED1\nRegistrar WHOIS Server: whois.registrar.test\n\nnic-hdl: SHARED1\norg: Example Owner\ne-mail: owner@example.test\n\nnsset: SHARED1\nnserver: ns1.example.test\n\nkeyset: SHARED1\ndnskey: 257 3 13 ABCDEF0123456789\n\nnic-hdl: UNRELATED\norg: Wrong Organization\ne-mail: tech@example.test\nurl: https://registrar.test/contact\n")
			default:
				t.Errorf("followed a registry referral or handle: %s", address)
			}
		}()
		return client, nil
	}
	r, err := c.Lookup(context.Background(), "example.at")
	if err != nil || len(destinations) != 1 || r.Registrant != "Example Owner" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" || strings.Join(r.Nameservers, ",") != "ns1.example.test" || r.DNSSEC != "signedDelegation" {
		t.Fatalf("%+v, %v, destinations=%v", r, err, destinations)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, "example.at", "SHARED1", "Example Owner", "owner@example.test", "whois.registrar.test", "registrar.test/contact")
}

func TestCNNICRestrictedNamesStopAtRegistry(t *testing.T) {
	for _, input := range []string{"net.cn", "home.cn", "https://net.cn/", "https://home.cn/path"} {
		t.Run(input, func(t *testing.T) {
			domain, err := NormalizeInput(input)
			if err != nil {
				t.Fatal(err)
			}
			c := fixtureClient(t, func(*http.Request) (*http.Response, error) {
				t.Fatal("unexpected HTTP request")
				return nil, nil
			})
			seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
			seedTLDs(t, c, map[string]string{"cn": "whois.cnnic.cn"})
			var destinations []string
			c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
				destinations = append(destinations, address)
				client, server := net.Pipe()
				go func() {
					defer server.Close()
					query, _ := bufio.NewReader(server).ReadString('\n')
					switch address {
					case "whois.cnnic.cn:43":
						if query != domain+"\r\n" {
							t.Errorf("namespace name was rewritten: %q", query)
						}
						fmt.Fprint(server, "the Domain Name you apply can not be registered online. Please consult your Domain Name registrar\n")
					default:
						t.Errorf("followed registrar suggestion: %s", address)
					}
				}()
				return client, nil
			}
			r, err := c.Lookup(context.Background(), domain)
			if err != nil || r.Domain != domain || r.Result != Restricted || len(destinations) != 1 {
				t.Fatalf("%+v, %v, destinations=%v", r, err, destinations)
			}
			assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, domain)
		})
	}
}
