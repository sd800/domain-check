// Package localweb serves the embedded, loopback-only registry interface.
package localweb

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
)

const Address = "domain-check.localhost:20771"
const URL = "http://" + Address

//go:embed assets/*
var assets embed.FS

type Lookup func(context.Context, string) (registry.Record, []string, error)

type Server struct {
	Version  string
	lookup   Lookup
	token    string
	index    []byte
	csp      string
	active   chan struct{}
	mu       sync.Mutex
	requests []time.Time
}

func New(version string, lookup Lookup) (*Server, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	if lookup == nil {
		lookup = func(ctx context.Context, domain string) (registry.Record, []string, error) {
			client := registry.NewClient(registry.DefaultCacheDir())
			var warnings []string
			client.Warn = func(code string) { warnings = append(warnings, code) }
			r, err := client.Lookup(ctx, domain)
			return r, warnings, err
		}
	}
	index, err := assets.ReadFile("assets/index.html")
	if err != nil {
		return nil, err
	}
	boot, err := assets.ReadFile("assets/boot.js")
	if err != nil {
		return nil, err
	}
	checksum := sha256.Sum256(boot)
	s := &Server{Version: version, lookup: lookup, token: hex.EncodeToString(nonce), active: make(chan struct{}, 2)}
	page := strings.Replace(string(index), "/*BOOT*/", string(boot), 1)
	for _, name := range []string{"style.css", "app.js", "logo.svg"} {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256(data)
		path := "/assets/" + name
		page = strings.ReplaceAll(page, path, path+"?v="+hex.EncodeToString(hash[:8]))
	}
	s.index = []byte(page)
	s.csp = "default-src 'none'; script-src 'self' 'sha256-" + base64.StdEncoding.EncodeToString(checksum[:]) + "'; style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'"
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", s.csp)
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	w.Header().Set("Vary", "Origin, Sec-Fetch-Site, Sec-Fetch-Mode, Sec-Fetch-Dest")
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	ip, ipErr := netip.ParseAddr(host)
	if err != nil || ipErr != nil || !ip.IsLoopback() || !allowedHost(r.Host) {
		http.Error(w, "Local access only", http.StatusForbidden)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
		http.Error(w, "Origin rejected", http.StatusForbidden)
		return
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "none" && site != "same-origin" {
		if (site != "cross-site" && site != "same-site") || !pageNavigation(r) {
			http.Error(w, "Cross-origin request rejected", http.StatusForbidden)
			return
		}
	}
	if r.URL.Path == "/api/lookup" {
		s.query(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch r.URL.Path {
	case "/api/health":
		w.Header().Set("X-Domain-Check", "registry-local-interface")
		writeJSON(w, 200, map[string]string{"version": s.Version})
	case "/api/session":
		writeJSON(w, 200, map[string]string{"token": s.token, "version": s.Version})
	default:
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/favicon.ico" {
			name := strings.TrimPrefix(r.URL.Path, "/")
			if name == "favicon.ico" {
				name = "assets/logo.svg"
			}
			// A flat, exact asset list avoids filesystem traversal and path cleaning.
			if strings.Contains(strings.TrimPrefix(name, "assets/"), "/") || strings.Contains(name, "..") || strings.HasSuffix(name, "index.html") || strings.HasSuffix(name, "boot.js") {
				http.NotFound(w, r)
				return
			}
			data, err := assets.ReadFile(name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			switch {
			case strings.HasSuffix(name, ".css"):
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			case strings.HasSuffix(name, ".js"):
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			case strings.HasSuffix(name, ".svg"):
				w.Header().Set("Content-Type", "image/svg+xml")
			case strings.HasSuffix(name, ".woff2"):
				w.Header().Set("Content-Type", "font/woff2")
			default:
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			}
			w.Header().Set("Cache-Control", "public, max-age=86400")
			if r.Method != http.MethodHead {
				_, _ = w.Write(data)
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if len(r.URL.RequestURI()) > 16384 {
			http.Error(w, "URL too long", http.StatusRequestURITooLong)
			return
		}
		if r.Method != http.MethodHead {
			_, _ = w.Write(s.index)
		}
	}
}

// External links may open the HTML document, but never APIs or embedded resources.
func pageNavigation(r *http.Request) bool {
	return (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		r.Header.Get("Sec-Fetch-Mode") == "navigate" &&
		r.Header.Get("Sec-Fetch-Dest") == "document" &&
		r.URL.Path != "/api" && !strings.HasPrefix(r.URL.Path, "/api/") &&
		r.URL.Path != "/assets" && !strings.HasPrefix(r.URL.Path, "/assets/") &&
		r.URL.Path != "/favicon.ico"
}

func allowedHost(host string) bool {
	return host == Address || host == "127.0.0.1:20771" || host == "[::1]:20771" || host == "localhost:20771"
}

func (s *Server) query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Domain-Check-Token")), []byte(s.token)) != 1 {
		http.Error(w, "Request rejected", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var input struct {
		Input string `json:"input"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
		writeJSON(w, 400, map[string]any{"error": map[string]string{"code": "invalid-domain"}})
		return
	}
	domain, err := registry.NormalizeInput(input.Input)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": map[string]string{"code": "invalid-domain"}})
		return
	}
	if !s.admit() {
		writeJSON(w, 429, map[string]any{"error": map[string]string{"code": "local-rate-limit"}})
		return
	}
	select {
	case s.active <- struct{}{}:
		defer func() { <-s.active }()
	default:
		writeJSON(w, 429, map[string]any{"error": map[string]string{"code": "local-busy"}})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	record, warnings, err := s.lookup(ctx, domain)
	if err != nil {
		code, detail := "query-failed", ""
		var query *registry.QueryError
		if errors.As(err, &query) {
			code, detail = query.Code, registry.Clean(query.Detail)
		}
		writeJSON(w, 502, map[string]any{"error": map[string]string{"code": code, "detail": detail}})
		return
	}
	values := map[string]any{"domain": registry.Clean(record.Domain), "idn": registry.UnicodeDomain(record.Domain), "registration": registry.Clean(record.Registration), "expiration": registry.Clean(record.Expiration), "changed": registry.Clean(record.Changed), "registrar": registry.Clean(record.Registrar), "registrant": record.RegistrantText(), "dnssec": registry.Clean(record.DNSSEC), "result": record.Result}
	values["tld"], values["registration_url"] = record.TLD, registry.Clean(record.RegistrationURL)
	values["admin"], values["organization"], values["role"] = registry.Clean(record.Admin), registry.Clean(record.AdminOrganization), registry.Clean(record.AdminRole)
	var address []string
	for _, line := range record.RegistrantAddress {
		if clean := registry.Clean(line); clean != "" {
			address = append(address, clean)
		}
	}
	values["address"] = strings.Join(address, "\n")
	values["city"], values["state"] = registry.Clean(record.RegistrantCity), registry.Clean(record.RegistrantState)
	values["postal_code"], values["country"] = registry.Clean(record.RegistrantPostalCode), registry.Clean(record.RegistrantCountry)
	if c := record.RegistrantContact; c != nil && !record.TLD {
		values["contact_details"] = true
		registrant, organization := c.DisplayNames()
		values["registrant"], values["registrant_organization"] = registry.Clean(registrant), registry.Clean(organization)
		values["city"], values["state_province"] = registry.Clean(c.City), registry.Clean(c.StateProvince)
		values["postal_code"], values["registrant_country"] = registry.Clean(c.PostalCode), registry.Clean(c.Country)
		for key, items := range map[string][]string{"street": c.Street, "address": c.Address, "contact_phones": c.Phones, "contact_emails": c.Emails} {
			clean := make([]string, 0, len(items))
			for _, item := range items {
				if text := registry.Clean(item); text != "" {
					clean = append(clean, text)
				}
			}
			if key == "street" || key == "address" {
				values[key] = strings.Join(clean, "\n")
			} else {
				values[key] = clean
			}
		}
	}
	dates := make(map[string]string, 3)
	for _, key := range []string{"registration", "expiration", "changed"} {
		dates[key] = output.DateText(output.Chinese, values[key].(string))
	}
	values["dates_zh"] = dates
	for name, items := range map[string][]string{"statuses": record.Statuses, "emails": record.RegistrantEmail, "nameservers": record.Nameservers, "phones": record.AdminPhones, "admin_emails": record.AdminEmails} {
		clean := make([]string, 0, len(items))
		for _, item := range items {
			if value := registry.Clean(item); value != "" {
				clean = append(clean, value)
			}
		}
		values[name] = clean
	}
	writeJSON(w, 200, map[string]any{"record": values, "warnings": warnings})
}

func (s *Server) admit() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	keep := s.requests[:0]
	for _, at := range s.requests {
		if now.Sub(at) < time.Minute {
			keep = append(keep, at)
		}
	}
	s.requests = keep
	if len(s.requests) >= 30 {
		return false
	}
	s.requests = append(s.requests, now)
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// Run binds IPv4 and IPv6 loopback only; it never listens on a LAN interface.
func Run(ctx context.Context, version string) error {
	service, err := New(version, nil)
	if err != nil {
		return err
	}
	listeners := make([]net.Listener, 0, 2)
	for _, address := range []string{"127.0.0.1:20771", "[::1]:20771"} {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			for _, l := range listeners {
				_ = l.Close()
			}
			return fmt.Errorf("cannot open local interface: %w", err)
		}
		listeners = append(listeners, listener)
	}
	server := &http.Server{Handler: service, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 50 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10, ErrorLog: log.New(io.Discard, "", 0)}
	done := make(chan error, 2)
	for _, listener := range listeners {
		go func() { done <- server.Serve(listener) }()
	}
	select {
	case <-ctx.Done():
	case err = <-done:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
