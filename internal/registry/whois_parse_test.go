package registry

import (
	"reflect"
	"strings"
	"testing"
)

// Synthetic records preserve registry formats without retaining live query
// inputs, contact details, or responses as fixtures.
func TestWHOISRegistryFormats(t *testing.T) {
	for _, test := range []struct {
		tld, response string
		want          Record
	}{
		{"ax", `domain...............: example.ax
status...............: Registered
created..............: 21.10.2000
expires..............: 21.10.2030
modified.............: 20.10.2026

Nameservers
nserver..............: ns1.example.ax

Holder
name.................: Example Owner
holder email.........: owner@example.ax

Registrar
registrar............: Example Registrar`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Changed: "2026-10-20", Statuses: []string{"Registered"}, Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.ax"}, Registrar: "Example Registrar", Nameservers: []string{"ns1.example.ax"}}},
		{"at", `domain: example.at
registrar: Example Registrar
registrant: HOLDER-1
tech-c: TECH-2
nserver: ns1.example.at 192.0.2.1
changed: 20261020 12:34:56
registered: 20001021 01:02:03
DNSSEC: Signed

personname: Example Person
organization: Example Organization
e-mail: owner@example.at
nic-hdl: HOLDER-1
changed: 20261021 15:00:00

personname: Wrong Technical Contact
organization: Wrong Organization
e-mail: tech@example.at
nic-hdl: TECH-2
changed: 20261021 16:00:00`, Record{Registration: "2000-10-21 01:02:03", Changed: "2026-10-20 12:34:56", Registrar: "Example Registrar", Registrant: "Example Organization", RegistrantEmail: []string{"owner@example.at"}, Nameservers: []string{"ns1.example.at"}, DNSSEC: "signedDelegation"}},
		{"ee", `Domain:
name: example.ee
status: ok (paid and in zone)
registered: 2000-10-21 01:02:03 +03:00
changed: 2026-10-20 12:34:56 +03:00
expire: 2030-10-21

Registrant:
name: Example Owner
email: owner@example.ee
changed: 2026-10-21 12:00:00 +03:00

Administrative contact:
name: Wrong Admin
email: admin@example.ee
changed: 2026-10-21 13:00:00 +03:00

Registrar:
name: Example Registrar
changed: 2026-10-21 14:00:00 +03:00

Name servers:
nserver: ns1.example.ee
changed: 2026-10-21 15:00:00 +03:00`, Record{Registration: "2000-10-20 22:02:03 UTC", Expiration: "2030-10-21", Changed: "2026-10-20 09:34:56 UTC", Statuses: []string{"ok (paid and in zone)"}, Registrar: "Example Registrar", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.ee"}, Nameservers: []string{"ns1.example.ee"}}},
		{"it", `Domain: example.it
Created: 2000-10-21 01:02:03
Last Update: 2026-10-20 12:34:56
Expire Date: 2030-10-21
Signed: no

Registrant
  Organization: Example Owner
  Created: 1990-01-01 00:00:00
  Last Update: 2026-10-21 14:00:00

Admin Contact
  Name: Wrong Admin
  Organization: Wrong Admin Organization
  Email: admin@example.it
  Created: 1991-01-01 00:00:00

Technical Contacts
  Organization: Wrong Technical Organization
  Email: tech@example.it
  Last Update: 2026-10-21 15:00:00

Registrar
  Organization: Example Registrar
  Name: REG-ID
  DNSSEC: yes

Nameservers
  ns1.example.it`, Record{Registration: "2000-10-21 01:02:03", Expiration: "2030-10-21", Changed: "2026-10-20 12:34:56", Registrant: "Example Owner", Registrar: "Example Registrar", Nameservers: []string{"ns1.example.it"}, DNSSEC: "unsigned"}},
		{"hk", `Domain Name Information:
Domain Name: example.hk
Domain Name Commencement Date: 21-10-2000
Expiry Date: 21-10-2030
DNSSEC:
  unsigned
Registrar Name: Example Registrar
Registrar Contact Information: registrar@example.hk

Registrant Contact Information:

Company English Name (It should be the same as the registered/corporation name on your Business Register Certificate or relevant documents): Example Owner
Company Chinese name: 示例机构
Email: owner@example.hk (Hotline: +0 12345)

Name Servers Information:

ns1.example.hk
ns2.example.hk

Status Information:
Domain Prohibit Status:`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Registrar: "Example Registrar", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.hk"}, Nameservers: []string{"ns1.example.hk", "ns2.example.hk"}, DNSSEC: "unsigned"}},
		{"jp", `Domain Information:
a. [Domain Name] EXAMPLE.JP
g. [Organization] Example Owner
p. [Name Server] ns1.example.jp
[Signing Key] 12345 8 2 ABCDEF0123456789
[Registered Date] 2000/10/21
[Last Update] 2026/10/20 12:34:56 (JST)
Contact Information:
[Name] Public Contact
[Email] owner@example.jp`, Record{Registration: "2000-10-21", Changed: "2026-10-20 03:34:56 UTC", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.jp"}, Nameservers: []string{"ns1.example.jp"}, DNSSEC: "signedDelegation"}},
		{"kz", `Domain Name............: example.kz
Registrant
Name...................: Example Person
Organization Name......: Example Owner
Email Address..........: owner@example.kz

Administrative Contact
Organization Name......: Wrong Organization
Email Address..........: admin@example.kz

Name Server............: ns1.example.kz
Domain created.........: 2000-10-21 01:02:03 (GMT+0:00)
Last modified..........: 2026-10-20 12:34:56 (GMT+0:00)
Expiration date........: 2030-10-21 01:02:03 (GMT+0:00)
Current Registrar......: Example Registrar`, Record{Registration: "2000-10-21 01:02:03 UTC", Expiration: "2030-10-21 01:02:03 UTC", Changed: "2026-10-20 12:34:56 UTC", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.kz"}, Registrar: "Example Registrar", Nameservers: []string{"ns1.example.kz"}}},
		{"bo", `Dominio: example.bo
TITULAR:
Nombre: Example Owner
Email: owner@example.bo

CONTACTO ADMINISTRATIVO:
Nombre: Wrong Admin
Email: admin@example.bo

CONTACTO TECNICO:
Nombre: Wrong Tech
Email: tech@example.bo

Fecha de registro: 2000-10-21
Fecha de vencimiento: 2030-10-21`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.bo"}}},
		{"by", `Domain name: example.by
Registrar: Example Registrar
Org: Example Owner
Email: owner@example.by
Update Date: 2026-10-20
Creation Date: 2000-10-21
Expiration Date: 2030-10-21`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Changed: "2026-10-20", Registrant: "Example Owner", Registrar: "Example Registrar", RegistrantEmail: []string{"owner@example.by"}}},
		{"mo", `Domain Name: example.mo
Record created on 2000-10-21 01:02:03
Record expires on 2030-10-21
Domain name servers:
ns1.example.mo 192.0.2.1
ns2.example.mo 2001:db8::1`, Record{Registration: "2000-10-21 01:02:03", Expiration: "2030-10-21", Nameservers: []string{"ns1.example.mo", "ns2.example.mo"}}},
		{"mx", `Domain Name: example.mx
Created On: 2000-10-21
Last Updated On: 2026-10-20
Expiration Date: 2030-10-21
Registrar: Example Registrar
Registrant:
  Name: Example Owner
  State: Example State
Administrative Contact:
  Name: Wrong Admin
  State: Wrong State
  Email: admin@example.mx
Name Servers:
  DNS: ns1.example.mx
DNSSEC DS Records:
  DS Record: 12345 8 2 ABCDEF0123456789`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Changed: "2026-10-20", Registrant: "Example Owner", Registrar: "Example Registrar", Nameservers: []string{"ns1.example.mx"}, DNSSEC: "signedDelegation", RegistrantContact: &RegistrantContact{Name: "Example Owner", StateProvince: "Example State"}}},
		{"tn", `Domain name.........: example.tn
Details:
Creation date.......: 21-10-2000 01:02:03 GMT+1
Domain status.......: Active
Registrar...........: Example Registrar
Owner Contact
Name................: Example Owner
Email...............: owner@example.tn
Administrativ contact
Name................: Wrong Admin
Email...............: admin@example.tn
Technical contact
Name................: Wrong Tech
Email...............: tech@example.tn
dnssec..............: signed
DNS servers
Name................: ns1.example.tn.
IP address
IP n° 1: 192.0.2.1
Name................: ns2.example.tn.`, Record{Registration: "2000-10-21 00:02:03 UTC", Statuses: []string{"Active"}, Registrar: "Example Registrar", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.tn"}, Nameservers: []string{"ns1.example.tn", "ns2.example.tn"}, DNSSEC: "signedDelegation"}},
		{"tr", `** Domain Name: example.tr
** Registrant:
Hidden upon user request
** Registrar:
NIC Handle: REG1
Organization Name: Example Registrar
** Domain Servers:
ns1.example.tr
** Additional Info:
Created on..............: 2000-Oct-21.
Expires on..............: 2030-Oct-21.
** Whois Server:
Last Update Time: 2026-10-21T00:00:00Z`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Registrar: "Example Registrar", Nameservers: []string{"ns1.example.tr"}}},
		{"gf", `domain: example.gf
changed: 2026-10-20 12:34:56.0
nameserver: ns1.example.gf
[holder]:
name: Example Owner
email: owner@example.gf
changed: 2026-10-21 10:00:00.0
[admin_c]:
name: Wrong Admin
email: admin@example.gf
changed: 2026-10-21 11:00:00.0`, Record{Changed: "2026-10-20 12:34:56", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.gf"}, Nameservers: []string{"ns1.example.gf"}}},
		{"lv", `[Domain]
Domain: example.lv
Status: active
[Holder]
Name: Example Owner
[Tech]
Name: Wrong Tech
Email: tech@example.lv
[Nservers]
Nserver: ns1.example.lv
[Whois]
Updated: 2026-10-21T00:00:00Z`, Record{Statuses: []string{"active"}, Registrant: "Example Owner", Nameservers: []string{"ns1.example.lv"}}},
		{"bn", `Domain Name: example.bn
Creation Date: 21-Oct-2000 01:02:03
Modified Date: 20-Oct-2026 12:34:56
Expiration Date: 21-Oct-2030 01:02:03
Registrant:

  Name: Example Owner
  Email: owner@example.bn
Administrative Contact:
  Name: Wrong Admin
  Email: admin@example.bn
Name Servers:
  ns1.example.bn`, Record{Registration: "2000-10-21 01:02:03", Expiration: "2030-10-21 01:02:03", Changed: "2026-10-20 12:34:56", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.bn"}, Nameservers: []string{"ns1.example.bn"}}},
		{"sm", `Domain Name: example.sm
Registration date: 21/10/2000
Owner:
Example Owner
Example Address
Email: owner@example.sm
Technical Contact:
Wrong Tech
Email: tech@example.sm
DNS Servers:
ns1.example.sm`, Record{Registration: "2000-10-21", Registrant: "Example Owner", RegistrantEmail: []string{"owner@example.sm"}, Nameservers: []string{"ns1.example.sm"}}},
		{"nc", `Domain: example.nc
Created on: 2000-10-21T01:02:03Z
Expires on: 2030-10-21T01:02:03Z
Last updated on: 2026-10-20T12:34:56Z
Domain server 1: ns1.example.nc
Domain server 2: ns2.example.nc`, Record{Registration: "2000-10-21 01:02:03 UTC", Expiration: "2030-10-21 01:02:03 UTC", Changed: "2026-10-20 12:34:56 UTC", Nameservers: []string{"ns1.example.nc", "ns2.example.nc"}}},
		{"gg", `Domain:
  example.gg
Registrant:
  Example Owner
Registrar:
  Example Registrar
Relevant dates:
  Registered on 21st October 2000 at 01:02:03.000
Registration status:
  Registered until cancelled
Name servers:
  ns1.example.gg
WHOIS lookup made on Wed, 21 Oct 2026 at 00:00:00`, Record{Registration: "2000-10-21 01:02:03", Registrant: "Example Owner", Registrar: "Example Registrar", Statuses: []string{"Registered until cancelled"}, Nameservers: []string{"ns1.example.gg"}}},
		{"cl", `Registrant name: Example Owner
Registrant organisation:
Registrar name: Example Registrar
Creation date: 2000-10-21 01:02:03 CLST
Expiration date: 2030-10-21 01:02:03 CLST
Name server: ns1.example.cl`, Record{Registration: "2000-10-21 04:02:03 UTC", Expiration: "2030-10-21 04:02:03 UTC", Registrant: "Example Owner", Registrar: "Example Registrar", Nameservers: []string{"ns1.example.cl"}}},
		{"se", `state: active
domain: example.se
holder: UNRESOLVED-ID
created: 2000-10-21
modified: 2026-10-20
expires: 2030-10-21
dnssec: signed delegation
registrar: Example Registrar`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Changed: "2026-10-20", Statuses: []string{"active"}, Registrar: "Example Registrar", DNSSEC: "signedDelegation"}},
		{"il", `query: example.il
domain: example.il
registrant: Example Owner
e-mail: owner@example.il
assigned: 20001021
validity: 21-10-2030
changed: registrar AT example.il 20001021 (Assigned)
changed: registrar AT example.il 20261020 (Changed)
admin e-mail: admin@example.il
changed: registrar AT example.il 20261021
tech e-mail: tech@example.il
registrar name: Example Registrar`, Record{Registration: "2000-10-21", Expiration: "2030-10-21", Changed: "2026-10-20", Registrant: "Example Owner", Registrar: "Example Registrar", RegistrantEmail: []string{"owner@example.il"}}},
	} {
		t.Run(test.tld, func(t *testing.T) {
			test.want.Domain = "example." + test.tld
			got, err := parseWHOIS(test.want.Domain, test.response)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got  %+v, %v\nwant %+v", got, err, test.want)
			}
		})
	}
}

func TestWHOISLinkedObjects(t *testing.T) {
	for _, tld := range []string{"mw", "mk", "ls", "ve", "ir", "at"} {
		t.Run(tld, func(t *testing.T) {
			domain := "example." + tld
			body := `domain: DOMAIN
registrant: OWNER1
admin-c: ADMIN1
nsset: NS1
keyset: KEY1
registrar: Example Registrar
registered: 2000-10-21
changed: 2026-10-20
expire: 2030-10-21

contact: ADMIN1
org: Wrong Admin
e-mail: admin@example.test
created: 1990-01-01
changed: 2026-10-21

contact: OWNER1
org: Example Owner
name: Example Person
e-mail: owner@example.test
created: 1991-01-01
changed: 2026-10-21

nsset: NS1
nserver: ns1.example.test 192.0.2.1
registrar: Wrong NS Registrar
created: 1992-01-01
changed: 2026-10-21

nsset: UNRELATED
nserver: wrong.example.test

keyset: KEY1
dnskey: 257 3 13 ABCDEF0123456789
created: 1993-01-01
changed: 2026-10-21
`
			body = strings.ReplaceAll(body, "DOMAIN", domain)
			if tld == "ir" || tld == "at" {
				body = strings.ReplaceAll(body, "contact:", "nic-hdl:")
			}
			r, err := parseWHOIS(domain, body)
			if err != nil || r.Registrant != "Example Owner" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" || strings.Join(r.Nameservers, ",") != "ns1.example.test" || r.Registrar != "Example Registrar" || r.Registration != "2000-10-21" || r.Expiration != "2030-10-21" || r.Changed != "2026-10-20" || r.DNSSEC != "signedDelegation" {
				t.Fatalf("%+v, %v", r, err)
			}
		})
	}
}

func TestWHOISUnknownHandlesAndUnrelatedContacts(t *testing.T) {
	for _, tld := range []string{"at", "ir", "mw", "mk", "ls", "ve"} {
		r, err := parseWHOIS("example."+tld, "domain: example."+tld+"\nregistrant: MISSING1\n\nnic-hdl: TECH1\norg: Wrong Technical Organization\ne-mail: tech@example.test\n")
		if err != nil || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
			t.Fatalf("%s: unresolved contact replaced with unrelated object: %+v, %v", tld, r, err)
		}
	}
}

func TestWHOISMultipleNameserversAndEmailValues(t *testing.T) {
	r, err := parseWHOIS("example.cn", "Domain Name: example.cn\nRegistrant Email: <owner@example.cn>, second@example.cn (Tel: +0 1234)\nName Server: NS1.EXAMPLE.CN., ns2.example.cn (192.0.2.1) 2001:db8::1 ns1.example.cn\n")
	if err != nil || strings.Join(r.Nameservers, ",") != "ns1.example.cn,ns2.example.cn" || strings.Join(r.RegistrantEmail, ",") != "owner@example.cn,second@example.cn" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISDoesNotAcceptIncompleteImplicitIdentity(t *testing.T) {
	for _, body := range []string{
		"Registrant name: Example Owner\nName server: ns1.example.cl",
		"Domain Name: other.cl\nRegistrant name: Example Owner\nRegistrar name: Example Registrar\nCreation date: 2000-01-01\nExpiration date: 2030-01-01\nName server: ns1.example.cl",
	} {
		if _, err := parseWHOIS("example.cl", body); err == nil {
			t.Fatal("accepted an incomplete or mismatched implicit-domain response")
		}
	}
}

func TestWHOISBlankRegistrarDoesNotConsumeDomainFields(t *testing.T) {
	for _, body := range []string{
		"Domain: example.at\nregistrar:\nregistrant: OWNER1\nregistered: 20001021 01:02:03\nchanged: 20261020 12:34:56\nDNSSEC: Signed\n\nnic-hdl: OWNER1\norg: Example Owner\ne-mail: owner@example.test",
		"Domain: example.at\nregistrar:\n\nDNSSEC: Signed\n\nDomain nameservers:\n  ns1.example.test\n\nCreation Date: 2000-10-21 01:02:03\nUpdated Date: 2026-10-20 12:34:56",
	} {
		r, err := parseWHOIS("example.at", body)
		if err != nil || r.Registration != "2000-10-21 01:02:03" || r.Changed != "2026-10-20 12:34:56" || r.DNSSEC != "signedDelegation" {
			t.Fatalf("%+v, %v", r, err)
		}
	}
}

func TestWHOISBundledAndAlternateDomainNames(t *testing.T) {
	for _, value := range []string{"EXAMPLE.HK Bundled Domain Name: 示例.hk /.香港", "example.hk (EXAMPLE.HK)"} {
		r, err := parseWHOIS("example.hk", "Domain Name: "+value+"\nName Server: ns1.example.test")
		if err != nil || len(r.Nameservers) != 1 {
			t.Fatalf("%+v, %v", r, err)
		}
	}
	for _, value := range []string{"other.hk (example.hk)", "https://example.hk", "example.hk other.hk"} {
		if _, err := parseWHOIS("example.hk", "Domain Name: "+value); err == nil {
			t.Fatalf("accepted mismatched or malformed domain %q", value)
		}
	}
}

func TestWHOISFooterDoesNotBecomeAvailability(t *testing.T) {
	for _, marker := range []string{">>> Last update of WHOIS database: 2026-10-21T00:00:00Z <<<", "Terms of Use:", "Copyright notice:"} {
		r, err := parseWHOIS("example.my", "Domain Name: example.my\nCreation Date: 2000-10-21\nName Server: ns1.example.test\n"+marker+"\nDomain names may be\nreserved.\nNo match for a different query.\n")
		if err != nil || r.Result != "" || r.Registration != "2000-10-21" {
			t.Fatalf("footer became availability: %+v, %v", r, err)
		}
	}
	if _, err := parseWHOIS("example.my", "Domain Name: example.my\nCreation Date: 2000-10-21\nDomain not found"); err == nil {
		t.Fatal("accepted a contradictory record outside a footer")
	}
}

func TestWHOISKeyMaterialAndNameserverTable(t *testing.T) {
	for _, body := range []string{
		"[Domain Name] EXAMPLE.JP\n[Signing Key] 12345 8 2 (\n  ABCDEF0123456789\n  ABCDEF0123456789 )\n",
		"Domain: example.jp\nKeys:\nflags: KSK protocol:3 algorithm:RSA_SHA256 pubKey:AbCdEf0123456789\n",
	} {
		r, err := parseWHOIS("example.jp", body)
		if err != nil || r.DNSSEC != "signedDelegation" {
			t.Fatalf("%+v, %v", r, err)
		}
	}
	r, err := parseWHOIS("example.mo", "Domain Name: example.mo\nDomain name servers:\n ---------------------------------------\nns1.example.mo\nns2.example.mo 2001:db8::1\n\nPlease visit our website for more information.\nwhois.example.test\n")
	if err != nil || strings.Join(r.Nameservers, ",") != "ns1.example.mo,ns2.example.mo" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISScopedRegistrarObject(t *testing.T) {
	r, err := parseWHOIS("example.sk", `Domain: example.sk
Created: 2000-10-21
Valid Until: 2030-10-21
Updated: 2026-10-20

Domain registrant: OWNER1
Name: Example Person
Organization: Example Owner
Email: owner@example.test
Created: 1990-01-01
Updated: 2026-10-21

Registrar: REG1
Name: Example Registrar
Organization: Example Registrar Organization
Email: registrar@example.test
Created: 1991-01-01
Updated: 2026-10-21
`)
	if err != nil || r.Registrant != "Example Owner" || r.Registrar != "Example Registrar Organization" || strings.Join(r.RegistrantEmail, ",") != "owner@example.test" || r.Registration != "2000-10-21" || r.Changed != "2026-10-20" || r.Expiration != "2030-10-21" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISFlatContactReferencesDoNotStartSections(t *testing.T) {
	r, err := parseWHOIS("example.ru", "domain: example.ru\nadmin-contact: https://registrar.test/contact\ncreated: 2000-10-21T01:02:03Z\npaid-till: 2030-10-21T01:02:03Z\n")
	if err != nil || r.Registration != "2000-10-21 01:02:03 UTC" || r.Expiration != "2030-10-21 01:02:03 UTC" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISStatusIsNotAnEmptyContactPlaceholder(t *testing.T) {
	r, err := parseWHOIS("example.be", "Domain: example.be\nStatus: NOT AVAILABLE\nRegistrant Name: DATA REDACTED\nRegistrant Email: Not Disclosed - Visit www.example.test\n")
	if err != nil || strings.Join(r.Statuses, ",") != "NOT AVAILABLE" || r.Registrant != "" || len(r.RegistrantEmail) != 0 || r.Result != "" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestWHOISMissingContactDoesNotEraseDomainRecord(t *testing.T) {
	r, err := parseWHOIS("example.mq", `domain: example.mq
changed: 2026-10-20 12:34:56.0
nameserver: ns1.example.test

[holder]:
NO OBJECT FOUND!
object: HOLDER1
type: PERSON

[admin_c]:
NO OBJECT FOUND!
object: ADMIN1
type: PERSON
`)
	if err != nil || r.Result != "" || r.Changed != "2026-10-20 12:34:56" || len(r.Nameservers) != 1 || r.Registrant != "" || len(r.RegistrantEmail) != 0 {
		t.Fatalf("missing contact became a missing domain: %+v, %v", r, err)
	}
}

func TestJPRSContactReferencesDoNotHideDomainDates(t *testing.T) {
	r, err := parseWHOIS("example.co.jp", "a. [Domain Name] EXAMPLE.CO.JP\ng. [Organization] Example Owner\nm. [Administrative Contact] OWNER1JP\nn. [Technical Contact]\n[Registered Date] 2000/10/21\n[Last Update] 2026/10/20 12:34:56 (JST)\n")
	if err != nil || r.Registration != "2000-10-21" || r.Changed != "2026-10-20 03:34:56 UTC" || r.Registrant != "Example Owner" {
		t.Fatalf("%+v, %v", r, err)
	}
}
