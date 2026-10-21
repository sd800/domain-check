package registry

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const eduCAUSERecord = `This Registry database contains ONLY .EDU domains.
The EDUCAUSE Whois database is authoritative for the .EDU domain.

Domain Name: EXAMPLE.EDU

Registrant:
 Example University
 The Board of Example University
 1 College Road
 Example City, EX 12345
 USA

Administrative Contact:
 Example Administrator
 Example University
 1 College Road
 Example City, EX 12345
 USA
 +1.5555550100
 admin@example.edu

Technical Contact:
 Example Network Operations
 Example University
 1 College Road
 Example City, EX 12345
 USA
 +1.5555550101
 technical@example.edu

Name Servers:
 NS1.EXAMPLE.TEST
 NS2.EXAMPLE.TEST

Domain record activated: 23-May-1985
Domain record last updated: 25-Jun-2026
Domain expires: 31-Jul-2027
`

const eduCAUSENoRecord = `Domain Name: example-unregistered.edu
The domain name you requested was not found in our database.

To check the availability of this domain name, please visit our website at https://registrar.test/
URL of the ICANN WHOIS Data Problem Reporting System: https://complaints.test/
>>> Last update of WHOIS database: 2026-10-21T00:00:00Z <<<
`

func TestEDUCAUSECompleteRegistryRecord(t *testing.T) {
	want := Record{Domain: "example.edu", Registrant: "Example University", Registration: "1985-05-23", Changed: "2026-06-25", Expiration: "2027-07-31", Nameservers: []string{"ns1.example.test", "ns2.example.test"}}
	for _, response := range []string{eduCAUSERecord, strings.ReplaceAll(eduCAUSERecord, "\n\n", "\n \t\n\n\n")} {
		r, err := parseWHOIS("example.edu", response)
		if err != nil || !reflect.DeepEqual(r, want) {
			t.Fatalf("%+v, %v; want %+v", r, err, want)
		}
	}
}

func TestEDUCAUSENegativeReplyAndSubjectValidation(t *testing.T) {
	r, err := parseWHOIS("example-unregistered.edu", eduCAUSENoRecord)
	if err != nil || !reflect.DeepEqual(r, Record{Domain: "example-unregistered.edu", Result: NoRecord}) {
		t.Fatalf("negative reply became an empty positive record: %+v, %v", r, err)
	}
	if r, err := parseWHOIS("other.edu", eduCAUSENoRecord); err == nil || r.Result != "" {
		t.Fatalf("mismatched negative reply became availability: %+v, %v", r, err)
	}
}

func TestEDUCAUSEDatePrecisionAndFallback(t *testing.T) {
	for input, want := range map[string]string{
		"23-May-1985": "1985-05-23", "3-MAY-1985": "1985-05-03",
		"2026-10-21T12:34:56Z": "2026-10-21 12:34:56 UTC",
		"before 1985":          "before 1985", "31-Feb-2026": "31-Feb-2026",
	} {
		if got := formatWHOISTime("edu", input); got != want {
			t.Fatalf("%s: got %q; want %q", input, got, want)
		}
	}
}

func TestEDUCAUSELookupStopsAtRegistry(t *testing.T) {
	c := fixtureClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("WHOIS website or registrar URL was fetched")
		return nil, nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	var destinations []string
	c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
		destinations = append(destinations, address)
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			query, _ := bufio.NewReader(server).ReadString('\n')
			switch address {
			case "whois.iana.org:43":
				if query != "edu\r\n" {
					t.Errorf("unexpected suffix query %q", query)
				}
				fmt.Fprint(server, "domain: EDU\nwhois: whois.educause.edu\n")
			case "whois.educause.edu:43":
				if query != "example-unregistered.edu\r\n" {
					t.Errorf("unexpected registry query %q", query)
				}
				fmt.Fprint(server, eduCAUSENoRecord)
			default:
				t.Errorf("followed a registrar or website reference: %s", address)
			}
		}()
		return client, nil
	}
	r, err := c.Lookup(context.Background(), "example-unregistered.edu")
	if err != nil || r.Result != NoRecord || len(destinations) != 2 {
		t.Fatalf("%+v, %v, destinations=%v", r, err, destinations)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "whois/edu.json"}, "example-unregistered.edu", "registrar.test", "complaints.test")
}
