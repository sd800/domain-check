# Technical details

Domain Check from Registries is a macOS CLI written in Go, with an optional local browser interface. Its executable is `domain`. Both interfaces share IANA service discovery, local response parsing, and English or Simplified Chinese presentation.

## Lookup flow

1. Parse one command or one domain/URL, with optional language selectors. Introduction, help, version, setup, and standalone language commands finish without registry discovery.
2. Validate a domain and convert internationalized labels to lowercase ASCII/Punycode. For URL input, discard credentials, port, path, query, and fragment locally, then select the registry-level registrable domain. The pasted URL is never fetched.
3. For a single-label TLD, query `https://rdap.iana.org/domain/<tld>` directly and skip discovery. For an ordinary domain, load the complete IANA `dns.json` document and select the longest matching suffix. Prefer an advertised HTTPS endpoint. If IANA advertises RDAP, use it even when it fails; failure never triggers a WHOIS fallback.
4. When no RDAP endpoint is advertised, obtain the active suffix's WHOIS hostname from the complete IANA `tlds.json` document and make one TCP port 43 query. Its `rdap_base_urls` fields do not override `dns.json`.
5. Validate the response's domain identity and extract fields. Use only embedded entities and objects already present in the same response. Never fetch registrar referrals, entity links, contact URLs, or unresolved handles.
6. Sanitize remote values and write the standardized layout. Missing and redacted values are omitted. A service error never becomes an availability claim.

The dedicated education-registry interface and DNS fallback are described in the README's [privacy policy](../README.md#privacy-policy). The fallback provides public nameserver records when the registry interface is unavailable.

## Domain and URL handling

Command names are recognized before query normalization. Bare commands keep priority; a dotted name such as `.setup` explicitly selects a TLD lookup. Query normalization ignores one leading dot for both TLDs and ordinary domains, so `.github.io` and `github.io` share the same identity. Multiple leading dots are rejected. Domain validation accepts a trailing dot and enforces DNS label and length rules. IP literals, embedded commands, URL components in a domain identity, and control characters are rejected. Unicode input uses IDNA lookup mapping. Joined emoji are supported through complete validated pictographic grapheme clusters, with registration eligibility determined by each registry.

Direct domain input keeps its normalized identity. URL input uses the ICANN portion of the bundled public suffix data to find the registry-level domain. Private hosting suffixes do not become registry boundaries: `https://user.github.io/` queries `github.io`. A multi-label namespace such as `net.cn` may itself be queried.

The first output row contains the ASCII domain. When a safe round-trip Unicode representation exists, an `IDN` / `国际化域名` row follows. The ASCII name remains the primary identity alongside its validated Unicode display. CLI values use no hyperlink escape sequences. Browser values use text nodes, and format detection is disabled for URLs, email addresses, telephone numbers, dates, and addresses.

## Response interpretation

| Output | Extraction rule |
| --- | --- |
| Domain | Must agree with the requested identity; conflicting declarations are rejected. |
| Registration / expiration / last changed | Domain events and domain-scoped WHOIS dates; contact and database-refresh timestamps are excluded. |
| Registrar | Explicit registrar role or registry field. |
| Status | Registry domain status, preserving source vocabulary. |
| Registrant | Disclosed registrant organization or name; administrative, technical, and billing contacts do not fill missing ownership data. |
| Registrant email | Only registrant-scoped disclosed addresses. |
| Nameservers | Disclosed delegation names, with WHOIS glue addresses separated from hostnames. |
| DNSSEC | Explicit domain signing information or sufficient DS/DNSKEY material; registrar capabilities are not domain signing status. |

RDAP parsing consumes domain events and embedded jCard entities. WHOIS parsing tokenizes labeled and multiline records once, identifies object and contact scopes, and applies shared field aliases plus observed registry-family syntax rules. Registry-operated names may have exceptional minimal records, so format verification also uses ordinary benign domains and explicit negative replies.

Dates retain source precision: English uses `YYYY-MM-DD` for dates and `YYYY-MM-DD HH:MM:SS` for timestamps; Chinese uses `YYYY 年 M 月 D 日` with the supplied time appended. The CLI and web interface share the same date presentation function. Explicitly zoned timestamps are converted to UTC and gain ` UTC`; unzoned timestamps retain their wall time. Approximate or unrecognized date text retains the registry's original wording.

RDAP `404` is reported as no registry record. Registration availability and restrictions are reported separately through explicit WHOIS statements such as not registered, reserved, not allowed, or restricted. Access denial, rate limits, timeouts, malformed data, and unknown syntax remain operational errors.

## TLD allocation records

Single-label queries use IANA RDAP as the authoritative root registry, including IDN TLDs. Only the fixed IANA endpoint may receive the query or an allowed redirect; alternate RDAP services and entity links in the response are never followed. These lookups neither load discovery caches nor save responses.

TLD status and registrant details share one output group with no intervening blank line or divider. The TLD layout shows domain registration and last-changed events, status, registrant, address, registration URL, administrative contact, nameservers, and DNSSEC. An entity with RDAP `roles: ["administrative"]` supplies the administrative block; the jCard `role` property supplies only the separate Role row. Name, organization, phone, and email values remain within that selected contact. Voice and fax numbers use repeated Phone rows. Registrant addresses prefer the jCard address label, retaining its line structure, with structured postal components as a fallback. The registration URL is the domain object's related link titled Registration URL, displayed only when it is an HTTP(S) URL. All displayed values are text; no link is fetched.

IANA HTTP 404 produces No IANA TLD record, without inferring registration availability. A successful inactive or reserved object retains its disclosed status and fields. Access denials, rate limits, service failures, malformed objects, and mismatched identities remain errors, with no WHOIS fallback.

## Presentation and commands

Label width is measured in terminal display cells. The content column follows the longest label's colon by three spaces. `DNSSEC` does not determine the Chinese column. Continuations begin at the same content column; wrapping preserves grapheme clusters. Terminal dimensions take precedence over `COLUMNS`, with an 80-column fallback.

`en`, `zh`, and `auto-lang` accept zero, one, or two leading hyphens. A standalone language command saves and confirms the default. With a domain, URL, help, version, update, or setup, the selection applies only to that invocation and bypasses reading the saved language. The last selector wins. The stored automatic-language value remains `auto` for compatibility; the command is `auto-lang`, and bare `auto` queries the `.auto` TLD. Device language supplies the initial default; explicit locale overrides and fallback behavior are documented in [Usage](../README.md#usage).

## Local browser interface

`domain setup`, `domain -setup`, and `domain --setup` open the same two-step menu. Choices appear in terminal cyan; other menu text keeps the normal terminal color. Redirected output remains plain, and `NO_COLOR` or `TERM=dumb` disables highlighting. The first step controls the web interface; the second controls automatic startup at macOS login. The current user's LaunchAgent runs the installed executable with a fixed internal server argument. Setup uses `/bin/launchctl` directly, with separate arguments and XML-escaped plist values. It does not invoke a shell, require administrator privileges, or modify the hosts file.

The server binds both `127.0.0.1:20771` and `[::1]:20771`. Browsers resolve the reserved `.localhost` hostname to loopback, making the interface available at `http://domain-check.localhost:20771`. A port conflict leaves the previous settings intact and reports a setup error.

HTML, CSS, JavaScript, and the text-free SVG logo are embedded in the executable. Content fingerprints in asset URLs keep cached assets consistent after an update. The title prefers the locally installed Segoe UI, with Arial and sans-serif fallbacks. Body text uses the system Arial and Chinese sans-serif stacks, including PingFang and Microsoft YaHei, at a 15px base size with normal letter spacing. Field labels are 14px and the copyright is 11px. The blue palette adapts to light and dark appearance. Search-box focus border and shadow changes are inactive; their CSS is retained in a comment for future adjustment. No remote font or asset is loaded.

Before first paint, a small CSP-hashed script applies the saved theme and language. Browser language and system appearance supply the initial defaults. Queries use `fetch` without navigation; the current record stays in place while another query is pending and is replaced atomically on success. Field columns retain a fixed alignment across languages, with a wider gap between labels and values. Long records scroll with the page. Submitting an empty input clears the result and returns to the initial view, cancelling any pending lookup. Only the theme and language are stored in browser local storage, separately from CLI preferences.

A direct route such as `/example.com`, `/.com`, `/.github.io`, or `/https%3A//www.google.com` decodes into the search input. The server avoids automatic path-cleaning redirects, preserving embedded URL separators. Input is submitted in a bounded JSON POST to the same-origin API and passes through the shared domain/URL normalizer before registry access. The browser does not load the pasted URL. After a successful query the address returns to `/` without creating a new history entry. Reloading starts with an empty result; responses and query inputs are not persisted.

The API requires a per-process random token, a permitted Host, loopback client address, and same-origin browser request metadata. It accepts at most two simultaneous lookups and thirty starts per minute, with a 45-second query deadline. The shared registry client retains its destination validation and registry-only routing. HTTP access and error logging are disabled; the LaunchAgent directs process output to `/dev/null`.

Turning off automatic startup keeps the service loaded for the current login session. Turning it on installs the plist in `~/Library/LaunchAgents`; turning the web service off unloads the job and removes both possible plists. Repeating an enabled setting reuses the existing registration without rewriting the plist or saved settings. The installer and `make install` use `launchctl kickstart` when a loaded service needs the updated executable; they do not unload, bootstrap, enable, or rewrite its registration. An already-current service stays running, and an unloaded service stays unloaded until explicit setup. Moving the executable or changing its launch configuration is applied through `domain setup`. Ordinary CLI lookups, introduction, help, and version never manage background registrations.

## Persistent state

| macOS path | Content |
| --- | --- |
| `~/Library/Caches/domain-check/dns.json` | Complete IANA RDAP bootstrap document. |
| `~/Library/Caches/domain-check/tlds.json` | Complete IANA TLD registry document. |
| `~/Library/Application Support/domain-check/config.json` | CLI language preference. |
| `~/Library/Application Support/domain-check/web.json` | Web enabled and start-at-login choices. |
| `~/Library/Application Support/domain-check/web-agent.plist` | Current-session LaunchAgent when login startup is off. |
| `~/Library/LaunchAgents/org.songming.domain-check.web.plist` | LaunchAgent when login startup is on. |
| Browser local storage | Web theme and language choices only. |

`DOMAIN_CACHE_DIR` and `DOMAIN_CONFIG_DIR` override the respective directories. Neither interface writes queried domains, pasted URLs, responses, or lookup history. Browser and shell history follow their own settings.

Cache age is checked only when a lookup needs the document. One calendar month after the last successful fetch, that lookup attempts a refresh, clamping month-end dates. A validated stale document may be reused with a warning after a failed automatic refresh; its age is not reset. Invalid documents are never routing evidence.

`domain update` validates both complete downloads before replacing either file. Each file is replaced atomically; a second save failure preserves the successful first refresh and reports the incomplete update. The optional web service refreshes IANA data only in response to a lookup. There are no scheduled updates or per-suffix cache files.

## Code organization

| Package / path | Responsibility |
| --- | --- |
| [`cmd/domain`](../cmd/domain) | Executable entry point. |
| [`internal/cli`](../internal/cli) | Arguments, locale, preferences, exit status, and terminal width. |
| [`internal/localweb`](../internal/localweb) | Embedded browser interface, local HTTP API, and current-user service management. |
| [`internal/registry`](../internal/registry) | Input normalization, IANA discovery, registry transports, and response parsing. |
| [`internal/output`](../internal/output) | Aligned bilingual records and grapheme-aware wrapping. |
| [`internal/safeio`](../internal/safeio) | Bounded reads of regular local data files. |
| [`scripts`](../scripts) | Installation, packaging, and installer security checks. |

Security controls and protocol details are maintained in [security.md](security.md). Coverage and service availability are described in [domains.md](domains.md).

## Build and release

Use Go 1.27.3 or newer. Release executables require macOS 13 (Ventura) or later. `go.mod` sets the minimum toolchain, and `go.sum` records dependency checksums. GitHub workflows use that version and run tests and vulnerability scanning before publishing. The CLI uses Go's standard network and JSON packages with bounded readers; specialized dependencies cover IDNA/public suffixes, selected legacy encodings, terminal sizing, and Unicode grapheme layout.

`make build` creates `build/bin/domain`. `make dist` creates CGO-free macOS Apple Silicon and Intel archives using `-trimpath` and stripped symbols, plus SHA-256 checksums. Each archive contains the executable, README, changelog, MIT License, version file, and this documentation directory. The standalone installer verifies the matching archive before replacing the executable. Only that explicitly invoked installer configures a standard zsh PATH entry; the CLI and `make install` install no shell behavior.

`VERSION` supplies the embedded calendar version `YY.M.D.NUM`. Release commits contain exactly `Release YY.M.D.NUM`; the matching `vYY.M.D.NUM` tag publishes release assets with an empty description. See [quality-assurance.md](quality-assurance.md#required-checks) for development checks.
