package localweb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sd800/domain-check/internal/registry"
)

func request(s *Server, method, path, body string) *http.Request {
	r := httptest.NewRequest(method, URL+path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:50123"
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Domain-Check-Token", s.token)
		r.Header.Set("Origin", URL)
	}
	return r
}

func TestWebLookupUsesNormalizedDomainAndNoRecordStorage(t *testing.T) {
	var inputs []string
	s, err := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		inputs = append(inputs, domain)
		return registry.Record{Domain: domain, Registrant: "Owner\x1b[2J\u202e", Nameservers: []string{"ns1.example.test"}}, []string{"stale-bootstrap"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r := request(s, "POST", "/api/lookup", `{"input":"https://user:password@www.example.com/path?private=1"}`)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 || len(inputs) != 1 || inputs[0] != "example.com" {
		t.Fatalf("%d %s, %v", w.Code, w.Body.String(), inputs)
	}
	if strings.ContainsAny(w.Body.String(), "\x1b\u202e") || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "private=1") {
		t.Fatal("unsafe output")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("lookup was cacheable")
	}
	if strings.Contains(string(s.index), "Owner") || strings.Contains(string(s.index), "example.com") {
		t.Fatal("lookup retained in page")
	}
}

func TestWebOriginHostTokenAndRemoteBoundary(t *testing.T) {
	var calls atomic.Int32
	s, _ := New("test", func(context.Context, string) (registry.Record, []string, error) {
		calls.Add(1)
		return registry.Record{}, nil, nil
	})
	for _, mutation := range []func(*http.Request){
		func(r *http.Request) { r.RemoteAddr = "192.168.1.4:5000" },
		func(r *http.Request) { r.Host = "attacker.test:20771" },
		func(r *http.Request) { r.Host = "domain-check.localhost.attacker.test:20771" },
		func(r *http.Request) { r.Header.Set("Origin", "https://attacker.test") },
		func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
		func(r *http.Request) { r.Header.Del("X-Domain-Check-Token") },
		func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") },
	} {
		r := request(s, "POST", "/api/lookup", `{"input":"example.com"}`)
		mutation(r)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("accepted request: %d", w.Code)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("rejected request reached registry")
	}
}

func TestWebInvalidAndOversizedInputDoNotQuery(t *testing.T) {
	s, _ := New("test", func(context.Context, string) (registry.Record, []string, error) {
		t.Fatal("invalid input reached registry")
		return registry.Record{}, nil, nil
	})
	for _, body := range []string{`{"input":"example.com\r\nother.test"}`, `{"input":"-h.example"}`, `{"input":"example.com","server":"attacker.test"}`, `{"input":"example.com"}{}`, `{"input":"` + strings.Repeat("x", 17000) + `"}`} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request(s, "POST", "/api/lookup", body))
		if w.Code != 400 {
			t.Fatalf("%d", w.Code)
		}
	}
}

func TestDeepLinksAssetsAndSecurityHeaders(t *testing.T) {
	s, _ := New("test", nil)
	for _, path := range []string{"/", "/example.com", "/cn", "/.com", "/.github.io", "/https%3A//www.google.com", "/https%3A%2F%2Fwww.example.com%2Fpath"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request(s, "GET", path, ""))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Domain Check") || strings.Contains(w.Body.String(), "/*BOOT*/") {
			t.Fatalf("deep link failed: %s %d", path, w.Code)
		}
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") || w.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatal("missing security headers")
		}
	}
	for _, path := range []string{"/assets/../server.go", "/assets/index.html", "/api/missing"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request(s, "GET", path, ""))
		if w.Code != 404 {
			t.Fatal(path, w.Code)
		}
	}
	for _, path := range []string{"/assets/logo.svg", "/assets/style.css", "/assets/app.js", "/favicon.ico"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request(s, "GET", path, ""))
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatal(path, w.Code)
		}
	}
	logo, _ := assets.ReadFile("assets/logo.svg")
	if strings.Contains(string(logo), "<text") {
		t.Fatal("logo contains text")
	}
}

func TestWebTLDFieldsAreSanitizedAndNamesNormalized(t *testing.T) {
	const payload = "text\x1b]8;;https://evil.test\x07\u202e"
	s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		return registry.Record{Domain: domain, TLD: !strings.Contains(domain, "."), Admin: payload, AdminOrganization: payload, AdminRole: payload, AdminPhones: []string{payload}, AdminEmails: []string{payload}, RegistrationURL: "http://registry.example.test/", RegistrantAddress: []string{"One Street", payload, "Country"}, RegistrantCity: payload, RegistrantState: payload, RegistrantPostalCode: payload, RegistrantCountry: payload}, nil, nil
	})
	for input, want := range map[string]string{"cn": "cn", ".com": "com", ".github.io": "github.io", ".setup": "setup"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"`+input+`"}`))
		var result struct {
			Record struct {
				Domain, Admin, Organization, Role, Address string
				City, State, Country                       string
				Postal                                     string `json:"postal_code"`
				URL                                        string `json:"registration_url"`
				Phones, Emails                             []string
				AdminEmails                                []string `json:"admin_emails"`
				TLD                                        bool
			}
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 {
			t.Fatal(err, w.Code, w.Body.String())
		}
		r := result.Record
		if r.City != registry.Clean(payload) || r.State != registry.Clean(payload) || r.Postal != registry.Clean(payload) || r.Country != registry.Clean(payload) {
			t.Fatal("unsafe or missing address components", w.Body.String())
		}
		if r.Domain != want || r.TLD != !strings.Contains(want, ".") || r.Admin == "" || r.Organization == "" || r.Role == "" || len(r.Phones) != 1 || len(r.AdminEmails) != 1 || len(r.Emails) != 0 || r.URL != "http://registry.example.test/" || !strings.HasPrefix(r.Address, "One Street\n") || !strings.HasSuffix(r.Address, "\nCountry") || strings.ContainsAny(w.Body.String(), "\x1b\u202e") {
			t.Fatal(w.Body.String())
		}
	}
	if !strings.Contains(string(s.index), `name="format-detection"`) {
		t.Fatal("browser auto-link detection was not disabled")
	}
}

func TestSessionAndNegativeResult(t *testing.T) {
	s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		return registry.Record{Domain: domain, Result: registry.Reserved}, nil, nil
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "GET", "/api/session", ""))
	var session map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &session)
	if session["token"] != s.token || len(session["token"]) != 64 {
		t.Fatal("invalid token")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.com"}`))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"result":"reserved"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestWebConcurrencyAndRateLimits(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		started <- struct{}{}
		<-release
		return registry.Record{Domain: domain}, nil, nil
	})
	done := make(chan struct{}, 2)
	for range 2 {
		go func() {
			defer func() { done <- struct{}{} }()
			s.ServeHTTP(httptest.NewRecorder(), request(s, "POST", "/api/lookup", `{"input":"example.com"}`))
		}()
	}
	<-started
	<-started
	w := httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.com"}`))
	if w.Code != 429 {
		t.Fatal("concurrency unbounded")
	}
	close(release)
	<-done
	<-done
	for range 30 {
		s.admit()
	}
	if s.admit() {
		t.Fatal("rate unbounded")
	}
}

func TestUnknownErrorsDoNotExposeInputs(t *testing.T) {
	s, _ := New("test", func(context.Context, string) (registry.Record, []string, error) {
		return registry.Record{}, nil, io.ErrUnexpectedEOF
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.com"}`))
	if w.Code != 502 || strings.Contains(w.Body.String(), "example.com") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestWebLocalizedDatesPreserveEnglishAndSourcePrecision(t *testing.T) {
	s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		return registry.Record{Domain: domain, Registration: "1994-04-08 04:00:00 UTC", Expiration: "2027-05-23", Statuses: []string{"ok"}}, nil, nil
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.com"}`))
	var result struct {
		Record struct {
			Registration string            `json:"registration"`
			Dates        map[string]string `json:"dates_zh"`
			Statuses     []string          `json:"statuses"`
		} `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Record.Registration != "1994-04-08 04:00:00 UTC" || result.Record.Dates["registration"] != "1994 年 4 月 8 日 04:00:00 UTC" || result.Record.Dates["expiration"] != "2027 年 5 月 23 日" || result.Record.Dates["changed"] != "" || len(result.Record.Statuses) != 1 || result.Record.Statuses[0] != "ok" {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestWebEDURegistrantRetainsSafeLines(t *testing.T) {
	s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		return registry.Record{Domain: domain, Registrant: "Example University", RegistrantDetails: []string{"Network Operations", "1 College Road\x1b[2J\u202e", "USA"}}, nil, nil
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.edu"}`))
	var data struct {
		Record struct{ Registrant string }
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || w.Code != 200 {
		t.Fatal(err, w.Code)
	}
	if data.Record.Registrant != "Example University\nNetwork Operations\n1 College Road [2J\nUSA" {
		t.Fatal(data.Record.Registrant)
	}
}

func TestDetailedRegistrantProjectionKeepsNameOrganizationAndSafeText(t *testing.T) {
	const payload = "text\x1b]8;;https://evil.test\x07\u202e"
	s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
		return registry.Record{Domain: domain, Registrant: "legacy organization", RegistrantContact: &registry.RegistrantContact{Name: payload, Organization: "Example Organization", Address: []string{"One Street", "City: Keep this line"}, Street: []string{payload}, City: payload, StateProvince: payload, PostalCode: payload, Country: payload, Phones: []string{payload}, Emails: []string{payload}}}, nil, nil
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.us"}`))
	var response struct{ Record map[string]any }
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 {
		t.Fatal(err, w.Code)
	}
	r := response.Record
	if r["contact_details"] != true || r["registrant"] != registry.Clean(payload) || r["registrant_organization"] != "Example Organization" || r["address"] != "One Street\nCity: Keep this line" {
		t.Fatal(r)
	}
	for _, key := range []string{"street", "city", "state_province", "postal_code", "registrant_country"} {
		if r[key] != registry.Clean(payload) {
			t.Fatal(key, r[key])
		}
	}
	for _, key := range []string{"contact_phones", "contact_emails"} {
		values, ok := r[key].([]any)
		if !ok || len(values) != 1 || values[0] != registry.Clean(payload) {
			t.Fatal(key, r[key])
		}
	}
}

func TestDetailedRegistrantNameAndOrganizationPresence(t *testing.T) {
	for _, tc := range []struct {
		name, organization, wantRegistrant, wantOrganization string
	}{
		{"", "Example Organization", "Example Organization", ""},
		{"REDACTED FOR PRIVACY", "Example Organization", "Example Organization", ""},
		{"Alex Example", "", "Alex Example", ""},
		{"Alex Example", "Example Organization", "Alex Example", "Example Organization"},
		{"Example Organization", "Example Organization", "Example Organization", "Example Organization"},
	} {
		s, _ := New("test", func(_ context.Context, domain string) (registry.Record, []string, error) {
			return registry.Record{Domain: domain, RegistrantContact: &registry.RegistrantContact{Name: tc.name, Organization: tc.organization, City: "Example City"}}, nil, nil
		})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request(s, "POST", "/api/lookup", `{"input":"example.us"}`))
		var response struct{ Record map[string]any }
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 {
			t.Fatal(err, w.Code)
		}
		if response.Record["registrant"] != tc.wantRegistrant || response.Record["registrant_organization"] != tc.wantOrganization {
			t.Fatal(tc, response.Record)
		}
	}
}
