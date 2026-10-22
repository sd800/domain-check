package registry

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
)

const arpaRegistryRecord = `% IANA WHOIS server
% This query returned 1 object

domain: RESOLVER.ARPA
organisation: Example Infrastructure Authority

contact: administrative
name: Wrong Administrative Contact
e-mail: admin@example.test

contact: technical
organisation: Wrong Technical Organization
e-mail: technical@example.test

remarks: This domain is administered as part of the .ARPA zone
created: 2000-10-21
source: IANA
`

func TestARPAUndelegatedRecordPreservesMetadata(t *testing.T) {
	r, err := parseWHOIS("resolver.arpa", arpaRegistryRecord)
	if err != nil || r.Registrant != "Example Infrastructure Authority" || r.Registration != "2000-10-21" || len(r.RegistrantEmail) != 0 || len(r.Nameservers) != 0 || r.DNSSEC != "" || r.Registrar != "" || r.Expiration != "" || r.Changed != "" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestARPADelegatedRecordAndNoObjectReply(t *testing.T) {
	r, err := parseWHOIS("example.arpa", `domain: EXAMPLE.ARPA
organisation: Example Infrastructure Authority
nserver: NS1.EXAMPLE.TEST 192.0.2.1 2001:db8::1
ds-rdata: 12345 8 2 ABCDEF0123456789
remarks: Infrastructure delegation
created: 2000-10-21
changed: 2026-10-20
source: IANA
`)
	if err != nil || r.DNSSEC != "signedDelegation" || len(r.Nameservers) != 1 || r.Registration != "2000-10-21" || r.Changed != "2026-10-20" {
		t.Fatalf("%+v, %v", r, err)
	}
	r, err = parseWHOIS("example-unregistered.arpa", "% IANA WHOIS server\n% This query returned 0 objects.\n% You queried for example-unregistered.arpa but this server does not have any data.\n")
	if err != nil || r.Result != NoRecord {
		t.Fatalf("%+v, %v", r, err)
	}
	if _, err := parseWHOIS("child.resolver.arpa", arpaRegistryRecord); err == nil {
		t.Fatal("parent-zone record became a child-name result")
	}
}

func TestARPALookupStopsAtIANADomainRegistry(t *testing.T) {
	c := fixtureClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP or referral request")
		return nil, nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	seedTLDs(t, c, map[string]string{"arpa": "whois.iana.org"})
	calls := 0
	c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
		calls++
		if address != "whois.iana.org:43" {
			t.Fatalf("followed an external WHOIS reference: %s", address)
		}
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			query, _ := bufio.NewReader(server).ReadString('\n')
			switch query {
			case "resolver.arpa\r\n":
				fmt.Fprint(server, arpaRegistryRecord+"refer: whois.external.test\n")
			default:
				t.Errorf("unexpected WHOIS query: %q", query)
			}
		}()
		return client, nil
	}
	r, err := c.Lookup(context.Background(), "resolver.arpa")
	if err != nil || r.Registration != "2000-10-21" || calls != 1 {
		t.Fatalf("%+v, %v, calls=%d", r, err, calls)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, "resolver.arpa", "admin@example.test", "whois.external.test")
}
