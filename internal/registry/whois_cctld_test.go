package registry

import (
	"reflect"
	"strings"
	"testing"
)

const kyrgyzFixture = `Domain EXAMPLE.EDU.KG (ACTIVE)
Administrative Contact:
 PID: Secured
Technical Contact:
 PID: Secured
Billing Contact:
 PID: Secured
Record created: Mon Oct 28 09:46:51 2024
Record last updated on: Wed Sep  9 11:15:29 2026
Record expires on: Thu Oct 28 09:46:51 2027
Name servers in the listed order:
NS1.EXAMPLE.TEST
NS2.EXAMPLE.TEST
`

func TestKyrgyzHeaderDatesAndHiddenOwner(t *testing.T) {
	r, err := parseWHOIS("example.edu.kg", kyrgyzFixture)
	if err != nil || r.Registrant != "" || len(r.RegistrantEmail) != 0 || r.Registration != "2024-10-28 09:46:51" || r.Changed != "2026-09-09 11:15:29" || r.Expiration != "2027-10-28 09:46:51" || len(r.Nameservers) != 2 || !reflect.DeepEqual(r.Statuses, []string{"ACTIVE"}) {
		t.Fatal(r, err)
	}
	if _, err := parseWHOIS("other.edu.kg", kyrgyzFixture); err == nil {
		t.Fatal("accepted another domain's header")
	}
	if r, err := parseWHOIS("example.edu.kg", "Domain EXAMPLE.EDU.KG\nData not found. This domain is available for registration.\n"); err != nil || r.Result != NotRegistered {
		t.Fatal(r, err)
	}
}

const koreanFixture = `query : example.go.kr
# KOREAN(UTF8)
도메인이름 : example.go.kr
등록인 : 예제 기관
책임자 : Example Administrator
책임자 전자우편 : admin@example.test
등록일 : 2000. 01. 17.
최근 정보 변경일 : 2023. 10. 31.
사용 종료일 : 2031. 01. 17.
등록대행자 : Example Registrar(http://registrar.test)
DNSSEC : 미서명
1차 네임서버 정보
 호스트이름 : NS1.EXAMPLE.TEST
 IP 주소 : 192.0.2.1
2차 네임서버 정보
 호스트이름 : NS2.EXAMPLE.TEST
# ENGLISH
Domain Name : example.go.kr
Registrant : Example Institution
Administrative Contact(AC) : Example Administrator
AC E-Mail : admin@example.test
Registered Date : 2000. 01. 17.
Last Updated Date : 2023. 10. 31.
Expiration Date : 2031. 01. 17.
Authorized Agency : Example Registrar(http://registrar.test)
DNSSEC : unsigned
Host Name : ns1.example.test
Host Name : ns2.example.test
`

func TestKoreanFieldsDatesAndContactScope(t *testing.T) {
	for _, domain := range []string{"example.go.kr", "example.xn--3e0b707e"} {
		body := strings.ReplaceAll(koreanFixture, "example.go.kr", domain)
		r, err := parseWHOIS(domain, body)
		if err != nil || r.Registrant != "예제 기관" || len(r.RegistrantEmail) != 0 || r.Registrar != "Example Registrar" || r.Registration != "2000-01-17" || r.Changed != "2023-10-31" || r.Expiration != "2031-01-17" || r.DNSSEC != "unsigned" || len(r.Nameservers) != 2 {
			t.Fatal(domain, r, err)
		}
	}
	for _, tc := range []struct {
		text string
		want Result
	}{{"등록자격이 제한된 도메인이름입니다.", EligibilityRestricted}, {"상기 도메인이름은 등록되어 있지 않습니다.", NoRecord}, {"상기 호스트이름은 등록되어 있지 않습니다.", NoRecord}} {
		body := "query : example.kr\n# KOREAN(UTF8)\n" + tc.text + "\n"
		r, err := parseWHOIS("example.kr", body)
		if err != nil || r.Result != tc.want {
			t.Fatal(r, err)
		}
		if _, err := parseWHOIS("other.kr", body); err == nil {
			t.Fatal("accepted mismatched restriction/absence")
		}
	}
}

func TestIcelandEmbeddedHolderAndDatePrecision(t *testing.T) {
	body := `domain: example.is
registrant: OWNER-IS
tech-c: TECH-IS
created: February 7 2002
expires: February 7 2027
nserver: ns1.example.test

role: Example Institution
nic-hdl: OWNER-IS
e-mail: owner@example.test
created: September 30 2002

role: Technical Provider
nic-hdl: TECH-IS
e-mail: tech@example.test
created: February 18 2021
`
	r, err := parseWHOIS("example.is", body)
	if err != nil || r.Registrant != "Example Institution" || !reflect.DeepEqual(r.RegistrantEmail, []string{"owner@example.test"}) || r.Registration != "2002-02-07" || r.Expiration != "2027-02-07" || r.Changed != "" {
		t.Fatal(r, err)
	}
}

func TestBrazilOwnerContactAndLegacyDecoding(t *testing.T) {
	body := []byte("domain: example.br\nowner: Educa\xe7\xe3o Example\nowner-c: OWNER-BR\ntech-c: TECH-BR\ncreated: 20010102\nchanged: 20260102\n\nnic-hdl-br: OWNER-BR\nperson: Domain Office\ne-mail: owner@example.test\ncreated: 20020102\n\nnic-hdl-br: TECH-BR\nperson: Technical Office\ne-mail: tech@example.test\n")
	text, err := decodeWHOIS("whois.registro.br", body)
	if err != nil {
		t.Fatal(err)
	}
	r, err := parseWHOIS("example.br", text)
	if err != nil || r.Registrant != "Educação Example" || !reflect.DeepEqual(r.RegistrantEmail, []string{"owner@example.test"}) || r.Registration != "2001-01-02" || r.Changed != "2026-01-02" {
		t.Fatal(r, err)
	}
	if _, err := decodeWHOIS("whois.unknown.test", body); err == nil {
		t.Fatal("guessed a charset for an unknown service")
	}
	if text, err := decodeWHOIS("whois.fi", []byte("domain: example.fi\nholder: Esimerkki \xe4\xf6\n")); err != nil || !strings.Contains(text, "äö") {
		t.Fatal(text, err)
	}
}

func TestCCTLDReservationRestrictionAndAbsence(t *testing.T) {
	for _, tc := range []struct {
		domain, body string
		want         Result
	}{{"example.ar", "El dominio no se encuentra registrado en NIC Argentina\n", NoRecord}, {"example.th", "% No match found.\n", NoRecord}, {"example.au", "Reserved by Registry\nThis domain name is Reserved by the Registry.\n", Reserved}, {"example.br", "% reserved: trademark\n", Reserved}, {"ac.br", "% blocked_word:  CG\n", NotAllowed}, {"example.uz", `Sorry, but domain: "example.uz", not found in database`, NoRecord}} {
		r, err := parseWHOIS(tc.domain, tc.body)
		if err != nil || r.Result != tc.want {
			t.Fatal(tc, r, err)
		}
	}
	if _, err := parseWHOIS("other.uz", `Sorry, but domain: "example.uz", not found in database`); err == nil {
		t.Fatal("accepted a different negative identity")
	}
	for _, domain := range []string{"edu.id", "nic.uz"} {
		r, err := parseWHOIS(domain, "Domain Name: "+domain+"\nCreated On: 2014-08-16T04:34:35Z\nStatus: RESERVED\nName Server: ns1.example.test\n")
		if err != nil || r.Result != Reserved || r.Registration != "" || len(r.Nameservers) != 0 {
			t.Fatal(r, err)
		}
	}
	r, err := parseWHOIS("example.uz", "Domain Name: example.uz\nName Server: not.defined.\nName Server: ns1.example.test\n")
	if err != nil || !reflect.DeepEqual(r.Nameservers, []string{"ns1.example.test"}) {
		t.Fatal(r, err)
	}
}

func TestCCTLDDatePrecisionAndHolderLabels(t *testing.T) {
	for _, tld := range []string{"cr", "cz", "fi", "tz"} {
		if got := formatWHOISTime(tld, "5.10.2027 04:03:02"); got != "2027-10-05 04:03:02" {
			t.Fatal(tld, got)
		}
	}
	if got := formatWHOISTime("br", "19970711 #46903"); got != "1997-07-11" {
		t.Fatal(got)
	}
	if got := formatWHOISTime("ua", "2027-04-15 09:59:52+03"); got != "2027-04-15 06:59:52 UTC" {
		t.Fatal(got)
	}
	r, err := parseWHOIS("example.ac.th", "Domain Name: example.ac.th\nCreated date: 2000-01-02\nExp date: 2027-01-02\nDomain Holder Organization: Example University\nTech Contact: Example Technician\nTech Organization: Technical Provider\n")
	if err != nil || r.Registrant != "Example University" || r.Expiration != "2027-01-02" {
		t.Fatal(r, err)
	}
	if got := formatWHOISTime("pl", "not defined"); got != "" {
		t.Fatal(got)
	}
}

func TestNominetMultilineParentsAndExplicitNamespaceNotices(t *testing.T) {
	for _, domain := range []string{"office.gov.uk", "office.nhs.uk", "unit.mil.uk"} {
		parent := domain[strings.IndexByte(domain, '.')+1:]
		body := "    Domain name:\n        " + parent + "\n\n    Registrant:\n        Parent Institution\n\n    Name servers:\n        ns1.example.test\n"
		r, err := parseWHOIS(domain, body)
		if err != nil || !reflect.DeepEqual(r, Record{Domain: domain, Result: NotProvided}) {
			t.Fatal(r, err)
		}
		if _, err := parseWHOIS("different.example.uk", body); err == nil {
			t.Fatal("accepted an unrelated multiline object")
		}
	}
	for _, tc := range []struct {
		domain, body string
		want         Result
	}{
		{"sch.uk", "Error for \"sch.uk\".\n\nNominet is not the registry for this domain name.\n", NotProvided},
		{"example.sch.uk", "Error for \"example.sch.uk\".\n\nThis domain cannot be registered because it contravenes the Nominet UK\nnaming rules.  The reason is:\n  invalid format for a .sch.uk domain name.\n", NotAllowed},
		{"example.edu.uk", "Error for \"example.edu.uk\".\n\nThis domain cannot be registered because it contravenes the Nominet UK\nnaming rules.  The reason is:\n  the domain name contains too many parts.\n", NotAllowed},
		{"example.police.uk", "Error for \"example.police.uk\".\n\nThis domain cannot be registered because it contravenes the Nominet UK\nnaming rules.  The reason is:\n  the domain name contains too many parts.\n", NotProvided},
		{"unit.mil.uk", "The domains, and sub-domains, of mod.uk and mil.uk are for use by the\nUK Armed Forces and Ministry of Defence establishments and systems.\nCorrespondence relating to them should be sent to hostmaster@mod.gov.uk.\n", NotProvided},
	} {
		body := tc.body + "WHOIS lookup made at 12:00:00 22-Oct-2026\nCopyright and terms follow.\n"
		r, err := parseWHOIS(tc.domain, body)
		if err != nil || !reflect.DeepEqual(r, Record{Domain: tc.domain, Result: tc.want}) {
			t.Fatal(tc, r, err)
		}
		if _, err := parseWHOIS("different.example.uk", body); err == nil {
			t.Fatal("accepted a namespace notice for an unrelated identity")
		}
	}
}
