# Technical details

Domain Check from Registries is a macOS CLI written in Go, with an optional local browser interface. Its Terminal command is `domain`, backed by the `domain-check` executable. Both interfaces share IANA service discovery, local response parsing, and English or Simplified Chinese presentation.

## Lookup flow

1. Parse one command or one domain/URL, with optional language selectors. Introduction, help, version, setup, and standalone language commands finish without registry discovery.
2. Validate a domain and convert internationalized labels to lowercase ASCII/Punycode. For URL input, discard credentials, port, path, query, and fragment locally, then select the registry-level registrable domain. The pasted URL is never fetched.
3. For a single-label TLD, query `https://rdap.iana.org/domain/<tld>` directly and skip discovery. Delegated `.ac.uk` and `.gov.uk` names use their documented namespace registry services below. For WHOIS-only `.tw` domain names, select registry WHOIS directly from the complete IANA TLD data and skip RDAP bootstrap discovery. For other ordinary domains, load the complete IANA `dns.json` document and select the longest matching suffix. Prefer an advertised HTTPS endpoint. RDAP is the primary protocol. On an RDAP connection, service, or response failure, validate the complete TLD dataset and use the active suffix's registry WHOIS host when published. Successful responses, rate limits, blocked redirects, and cancellation do not trigger fallback.
4. When no RDAP endpoint is advertised, obtain the active suffix's WHOIS hostname from the complete IANA `tlds.json` document and make one TCP port 43 query. Its `rdap_base_urls` fields do not override `dns.json`.
5. Validate the response's domain identity and extract fields. Use only embedded entities and objects already present in the same response. Never fetch registrar referrals, entity links, contact URLs, or unresolved handles.
6. Sanitize remote values and write the standardized layout. Fields missing from or explicitly redacted in the registry response are omitted, except that an empty Registrant row introduces any disclosed subordinate contact fields. A service error never becomes an availability claim.

The dedicated education-registry interface and DNS fallback are described in the README's [privacy policy](../README.md#privacy-policy). The fallback provides public nameserver records when the registry interface is unavailable.

## Domain and URL handling

Command names are recognized before query normalization. Bare commands keep priority; a dotted name such as `.setup` explicitly selects a TLD lookup. Query normalization ignores one leading dot for both TLDs and ordinary domains, so `.github.io` and `github.io` share the same identity. Multiple leading dots are rejected. Domain validation accepts a trailing dot and enforces DNS label and length rules. IP literals, embedded commands, URL components in a domain identity, and control characters are rejected. Unicode input uses IDNA lookup mapping. Joined emoji are supported through complete validated pictographic grapheme clusters, with registration eligibility determined by each registry.

Direct domain input keeps its normalized identity. URL input uses the ICANN portion of the bundled public suffix data to find the registry-level domain. Private hosting suffixes do not become registry boundaries: `https://user.github.io/` queries `github.io`. A multi-label namespace such as `net.cn` may itself be queried.

The first output row contains the ASCII domain. When a safe round-trip Unicode representation exists, an `IDN` / `国际化域名` row follows. The ASCII name remains the primary identity alongside its validated Unicode display. CLI values use no hyperlink escape sequences. Browser values use text nodes, and format detection is disabled for URLs, email addresses, telephone numbers, dates, and addresses.

## Response interpretation

| Output | Extraction rule |
| --- | --- |
| Domain | A registration record must agree with the requested identity; parent-only namespace objects are reported separately and conflicting declarations are rejected. |
| Registration / expiration / last changed | Domain events and domain-scoped WHOIS dates; contact and database-refresh timestamps are excluded. |
| Registrar | Explicit registrar role or registry field. Ordinary-domain RDAP results append a disclosed two-letter country code from that same registrar entity; TLD results do not. |
| Status | Registry domain status, preserving source vocabulary. |
| Registrant | Disclosed registrant organization or name; administrative, technical, and billing contacts do not fill missing ownership data. |
| Registrant email | Only registrant-scoped disclosed addresses. |
| Registrant address | For `.gov` RDAP, the selected registrant's disclosed street, city, state, postal code, and country populate separate aligned rows. Missing and redacted components are omitted; free-form labels stay in Address without inferring components. TLD allocation addresses retain their existing multiline layout. |
| Nameservers | Disclosed delegation names, with WHOIS glue addresses separated from hostnames. |
| DNSSEC | Explicit domain signing information or sufficient DS/DNSKEY material; registrar capabilities are not domain signing status. |

For EDUCAUSE `.edu` WHOIS responses, the Registrant row retains the indented institution, department, and postal lines under one label. Continuation collection stops at the end of that block, a contact heading, or a recognized field; administrative and technical contacts remain separate. Redacted institutions do not promote address lines into a name. The block shares the parser's field limits and output sanitization.

For ordinary-domain RDAP, registrar country annotation reads the jCard address `cc` parameter (RFC 8605), falling back only to an address country component that itself contains two ASCII letters. The code is uppercased and belongs to the same selected registrar entity as the name. Full country names are not mapped, other contact countries are not substituted, and conflicting address codes are omitted. TLD allocation names and organizations remain unannotated.

Detailed registrant contacts retain separate name and organization fields: RDAP jCard `fn` supplies the name and `org` supplies the organization; WHOIS uses the corresponding registrant-scoped labels or the selected embedded owner object. Separately supplied names and organizations occupy their own rows even when their values match. An organization without a disclosed name appears as Registrant, with no standalone Organization row. Structured addresses populate Street, City, State/Province, Postal Code, and Country. WHOIS Address/Addr labels and free-form RDAP address labels remain opaque multiline Address values. Disclosed phone and email values belong to that same contact; fax and phone-extension fields do not become phone rows. Redacted values are omitted. The detailed view activates when an address component is disclosed and preserves established `.edu`, `.gov`, TLD, and dedicated registry address layouts.

Both languages place Postal Code between State/Province and Country. Simplified Chinese uses the concise labels 邮编, 电话, and 邮箱 for postal code, phone, and email in the detailed view.

Generic registry WHOIS handling covers both thin records and full registrant contacts. Thin records retain only the fields actually supplied; a WHOIS database refresh never fills a missing domain date. IANA `.int` owner organizations and repeated address lines form one disclosed owner block, ending before administrative and technical contacts. `DNSSEC DS Data` uses the same validated key-data handling as other DS labels.

Registry absence dialects include explicit no-information and queried-object-does-not-exist replies. Rate limits, unsupported interfaces, invalid query syntax, empty responses, and service retirement remain operational failures. A retirement notice does not imply domain availability or select a different endpoint.

RDAP parsing consumes domain events and embedded jCard entities. WHOIS parsing tokenizes labeled and multiline records once, identifies object and contact scopes, and applies shared field aliases plus observed registry-family syntax rules. Registry-operated names may have exceptional minimal records, so format verification also uses ordinary benign domains and explicit negative replies.

Dates retain source precision: English uses `YYYY-MM-DD` for dates and `YYYY-MM-DD HH:MM:SS` for timestamps; Chinese uses `YYYY 年 M 月 D 日` with the supplied time appended. The CLI and web interface share the same date presentation function. Explicitly zoned timestamps are converted to UTC and gain ` UTC`; unzoned timestamps retain their wall time. Approximate or unrecognized date text retains the registry's original wording.

RDAP `404` is reported as no registry record. Registration availability and restrictions are reported separately through explicit WHOIS statements such as not registered, reserved, not allowed, or restricted. Access denial, rate limits, timeouts, malformed data, and unknown syntax remain operational errors.

## Registry WHOIS fallback

Fallback eligibility uses an active TLD entry and its WHOIS hostname from IANA's complete dataset. Both country-code and generic TLDs qualify. When its WHOIS field is empty for `.uk`, the fixed Nominet-published `whois.nic.uk` service is used. It covers ASCII and internationalized suffixes. A missing hostname never causes a guessed WHOIS destination. The WHOIS destination is never taken from RDAP response links. TLD allocation queries remain at IANA RDAP; `.tw` is selected as WHOIS-only before RDAP discovery.

A connection failure, unsuccessful HTTP service response other than a rate limit, or rejected response format can use registry WHOIS. HTTP 429, blocked redirects, and cancelled or expired contexts stop the lookup. A successful RDAP record, including omitted fields, a parent-only disclosure, or HTTP 404, completes the lookup without another query. Discovery errors and missing WHOIS services preserve the RDAP error. If both protocols fail, diagnostics retain both failures. The parser uses the successful WHOIS response alone and never merges it with a failed RDAP response.

Registry-specific WHOIS handling includes KISA's Korean and English fields, date-only dotted dates, and explicit registration-eligibility notices; Kyrgyz header/status and unzoned weekday timestamps; ISNIC embedded owner contacts and month-name dates; Nominet multiline parent records and delegated-namespace notices; Registro.br owner/contact scope and Latin-1 replies; Finnish Latin-1 replies; explicit registry usage restrictions; and explicit reserved metadata or absence statements from the relevant registries. Administrative emails remain separate from registrant emails, and placeholders never become nameservers.

## TWNIC domain handling

Ordinary names ending in `.tw` use the WHOIS host published in the complete IANA TLD dataset, even when `dns.json` advertises RDAP. This explicit routing choice does not contact RDAP first and is not an automatic fallback. The TLD query `tw` still goes directly to IANA RDAP.

TWNIC parsing separates comma-delimited domain statuses, extracts email-shaped values only within the registrant section, recognizes its registration-service-provider field, and converts explicitly supplied UTC offsets while preserving date precision. Referrals and registration-service URLs remain response data and are never fetched.

A child `.edu.tw` query may receive only the `edu.tw` manager's record. A child `.gov.tw` query may receive only a government RWHOIS referral. The observed `.mil.tw` reply rejects the namespace query without disclosing registration data. These specific responses produce `Registry did not provide domain-specific registration data`, retaining the requested domain and making no availability claim. A real domain-specific record, an explicit absence, a reserved parent name, and service errors remain distinct.

## Delegated and special-purpose namespaces

Names beneath `.ac.uk` go directly to the Jisc namespace registry's `whois.ja.net:43` service. Names beneath `.gov.uk` go directly to `https://rdap.nominet.uk/gov.uk/`. These fixed routes apply before ordinary TLD discovery, rather than after a failed lookup. Querying the parent `ac.uk` or `gov.uk` itself still uses normal `.uk` discovery. These services are namespace registries; no registrar referral supplies or changes a destination. Redirects remain confined to the selected registry route and the same query.

Jisc's owner, registration agent, server list, renewal, entry-created, and entry-updated fields map to the standard layout. Ordinal weekday dates retain date-only precision. Postal lines and contact names do not replace the institution; a disclosed registrant email is read only from its contact block.

For child names in `.nhs.uk`, `.police.uk`, `.mod.uk`, `.mil.uk`, and `.parliament.uk`, a 404 specifically from Nominet's parent `.uk` interface means the parent registry did not provide independent registration data for the queried child. That response does not establish absence at the independently administered namespace. Successful matching records are parsed normally, access and transport failures remain errors, and parent-name queries retain ordinary behavior.

An RDAP or WHOIS object identifying only a known ICANN public namespace above the query produces `Registry did not provide domain-specific registration data`. The requested identity remains visible and all parent contact, date, and delegation fields are discarded. The observed RNIDS `.mil.rs` parent-only response has the same explicit handling. Unrelated identities, private hosting suffixes, contradictory objects, and negative statements about another name remain errors. A direct query for the namespace itself is interpreted as its own record.

JPRS's reserved category records can contain delegation nameservers; an explicit reserved state for the category itself remains Reserved. Exact usage-restriction responses observed at the `.ie`, `.nz`, and `.sx` registry interfaces remain restrictions. Query-syntax failures, rate limits, refusal, and temporary unavailability stay service errors. Registration eligibility alone never suppresses a disclosed record or determines a name's availability.

NIC Senegal's IANA-listed HTTPS interface returns a single-field `whois43response` JSON envelope. For `.sn`, that exact envelope is decoded locally and passed to the bounded WHOIS parser, with its escaped line separators normalized. French domain labels and holder/contact sections map to the standard fields; contact-modification dates do not replace the domain's last change. This compatibility rule uses the original HTTPS response and makes no port 43 or referral request.

## TLD allocation records

Single-label queries use IANA RDAP as the authoritative root registry, including IDN TLDs. Only the fixed IANA endpoint may receive the query or an allowed redirect; alternate RDAP services and entity links in the response are never followed. These lookups neither load discovery caches nor save responses.

TLD status follows the last-changed row in the date group with no intervening blank line or divider. If that date is undisclosed, status follows the remaining dates. The TLD layout shows domain registration and last-changed events, status, registrant, address, registration URL, administrative contact, nameservers, and DNSSEC. An entity with RDAP `roles: ["administrative"]` supplies the administrative block; the jCard `role` property supplies only the separate Role row. Name, organization, phone, and email values remain within that selected contact. Voice and fax numbers use repeated Phone rows. Registrant addresses prefer the jCard address label, retaining its line structure, with structured postal components as a fallback. The registration URL is the domain object's related link titled Registration URL, displayed only when it is an HTTP(S) URL. All displayed values are text; no link is fetched.

IANA HTTP 404 produces No IANA TLD record, without inferring registration availability. A successful inactive or reserved object retains its disclosed status and fields. Access denials, rate limits, service failures, malformed objects, and mismatched identities remain errors, with no WHOIS fallback.

## Presentation and commands

Label width is measured in terminal display cells. The content column follows the longest label's colon by three spaces. `DNSSEC` does not determine the Chinese column. Continuations begin at the same content column; wrapping preserves grapheme clusters. Terminal dimensions take precedence over `COLUMNS`, with an 80-column fallback.

`en`, `zh`, and `auto-lang` accept zero, one, or two leading hyphens. A standalone language command saves and confirms the default. With a domain, URL, help, version, update, setup, or uninstall, the selection applies only to that invocation and bypasses reading the saved language. The last selector wins. The stored automatic-language value remains `auto` for compatibility; the command is `auto-lang`, and bare `auto` queries the `.auto` TLD. Device language supplies the initial default; explicit locale overrides and fallback behavior are documented in [Usage](../README.md#usage).

## Uninstallation

`domain uninstall` and its one- or two-hyphen forms request explicit confirmation with a default of cancel. A complete affirmative line is required. Empty, negative, incomplete, oversized, failed, or interrupted input never authorizes removal. Language selection for this command is temporary, and no registry discovery is performed.

The command validates the resolved `domain-check` executable and its optional sibling `domain` link before changing the installation. It uses the current user's fixed launchd job, distinguishes an absent job from a failed service inspection, and stops a loaded job before deleting known service files. Malformed saved settings do not block removal. Cleanup uses directory-scoped filesystem handles and exact filenames, unlinks file symlinks without following their targets, preserves unknown entries and containing directories, and rechecks executable and command identities before removal. It never recursively removes configuration or cache directories. `make uninstall` invokes the same confirmed command through the installed executable. Source files, private development samples, shell PATH settings, and browser-stored display choices are retained.

## Local browser interface

`domain setup`, `domain -setup`, and `domain --setup` open the same two-step menu. Choices appear in terminal cyan; other menu text keeps the normal terminal color. Redirected output remains plain, and `NO_COLOR` or `TERM=dumb` disables highlighting. The first step controls the web interface; the second controls automatic startup at macOS login. The current user's LaunchAgent runs the installed executable with a fixed internal server argument. Setup uses `/bin/launchctl` directly, with separate arguments and XML-escaped plist values. It does not invoke a shell, require administrator privileges, or modify the hosts file.

The server binds both `127.0.0.1:20771` and `[::1]:20771`. Browsers resolve the reserved `.localhost` hostname to loopback, making the interface available at `http://domain-check.localhost:20771`. A port conflict leaves the previous settings intact and reports a setup error.

HTML, CSS, JavaScript, and the text-free SVG logo are embedded in the executable. Content fingerprints in asset URLs keep cached assets consistent after an update. The title prefers the locally installed Segoe UI, with Arial and sans-serif fallbacks. Body text uses the system Arial and Chinese sans-serif stacks, including PingFang and Microsoft YaHei, at a 15px base size with normal letter spacing. Field labels are 14px and the copyright is 11px. The blue palette adapts to light and dark appearance. Search-box focus border and shadow changes are inactive; their CSS is retained in a comment for future adjustment. No remote font or asset is loaded.

Before first paint, a small CSP-hashed script applies the saved theme and language. Browser language and system appearance supply the initial defaults. Queries use `fetch` without navigation; the current record stays in place while another query is pending and is replaced atomically on success. Query errors and service notices appear above the result; failed lookups retain the previous record. Field columns retain a fixed alignment across languages, with a wider gap between labels and values. Long records scroll with the page. Submitting an empty input clears the result and returns to the initial view, cancelling any pending lookup. Only the theme and language are stored in browser local storage, separately from CLI preferences.

A direct route such as `/example.com`, `/.com`, `/.github.io`, or `/https%3A//www.google.com` decodes into the search input. The server avoids automatic path-cleaning redirects, preserving embedded URL separators. Input is submitted in a bounded JSON POST to the same-origin API and passes through the shared domain/URL normalizer before registry access. The browser does not load the pasted URL. After a successful query the address returns to `/` without creating a new history entry. Reloading starts with an empty result; responses and query inputs are not persisted.

External links can open the HTML interface through top-level GET/HEAD navigation (`Sec-Fetch-Mode: navigate`, `Sec-Fetch-Dest: document`). This exception applies only to page routes; it does not permit cross-origin API access, iframe embedding, or resource inclusion. An explicitly mismatched Origin is still rejected. Asset cache keys vary by Origin and Fetch Metadata.

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
| `~/Library/LaunchAgents/localhost.domain-check.web.plist` | LaunchAgent when login startup is on. |
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
| [`web`](../web) | Buildless public product website and local static assets. |

Security controls and protocol details are maintained in [security.md](security.md). Coverage and service availability are described in [domains.md](domains.md).

## Public product website

The public website lives in `web/`, separately from the application's embedded local interface. Open `web/index.html` directly in a browser, or serve `web/` with any static HTTP server. All styles, scripts, and the shared logo use relative paths, so the same files work from a local folder or a GitHub Pages project path. No build, package installation, remote font, or JavaScript module loader is required.

The hosted homepage uses `https://domain-check.songming.org/` as its canonical URL. Requests for `/index.html` or `/index` run an early same-origin browser redirect to `/`, preserving the query string and fragment and replacing the alias in browser history. The guard applies only to HTTP(S), so direct local-file opening remains unchanged.

English and Simplified Chinese content is included in the document. A small startup script applies the saved language and system/light/dark appearance before styles render; preference changes reuse the existing page. Only these two display choices are saved. The page contains illustrative registration output and installation guidance; domain queries remain in the installed application. The copy button uses the same complete installation block as the README, with a local clipboard fallback.

The `Website` GitHub Actions workflow validates the site and its redirect behavior and publishes only `web/` to GitHub Pages when website files, its checker, the README, or the workflow change on `main`. To enable the repository once, select **Settings → Pages → Build and deployment → Source → GitHub Actions**. Subsequent deployments use the workflow automatically; **Run workflow** can also publish the current source. The workflow is independent of macOS executable builds.

## Build and release

Use Go 1.27.3 or newer. Release executables require macOS 13 (Ventura) or later. `go.mod` sets the minimum toolchain, and `go.sum` records dependency checksums. GitHub workflows use that version and run tests and vulnerability scanning before publishing. The CLI uses Go's standard network and JSON packages with bounded readers; specialized dependencies cover IDNA/public suffixes, selected legacy encodings, terminal sizing, and Unicode grapheme layout.

`make build` creates the `build/bin/domain-check` executable and a `domain` command link. Installation keeps this same arrangement in the selected binary directory: `domain` points to `domain-check`, while LaunchAgents use the resolved executable path. macOS builds use the ad-hoc signing identifier `localhost.domain-check`. `make dist` creates CGO-free macOS Apple Silicon and Intel archives using `-trimpath` and stripped symbols, plus SHA-256 checksums. Each `domain-check_<version>_darwin_<architecture>.tar.gz` archive contains the `domain-check` executable, README, changelog, MIT License, version file, and this documentation directory. The standalone installer verifies the matching archive before replacing the executable and command link. Only that explicitly invoked installer configures a standard zsh PATH entry; the CLI and `make install` install no shell behavior.

`VERSION` supplies the embedded calendar version `YY.M.D.NUM`. Release commits contain exactly `Release YY.M.D.NUM`; the matching `vYY.M.D.NUM` tag publishes release assets with an empty description. See [quality-assurance.md](quality-assurance.md#required-checks) for development checks.
