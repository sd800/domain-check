# Technical details

Domain Check from Registries is a macOS CLI written in Go. Its executable is `domain`. It discovers registry services from IANA, extracts registration details locally, and renders English or Simplified Chinese output.

## Lookup flow

1. Parse one command or one domain/URL, with optional language selectors. Introduction, help, version, and standalone language commands finish without registry discovery.
2. Validate a domain and convert internationalized labels to lowercase ASCII/Punycode. For URL input, discard credentials, port, path, query, and fragment locally, then select the registry-level registrable domain. The pasted URL is never fetched.
3. Load the complete IANA `dns.json` document and select the longest matching suffix. Prefer an advertised HTTPS endpoint. If IANA advertises RDAP, use it even when it fails; failure never triggers a WHOIS fallback.
4. When no RDAP endpoint is advertised, obtain the active suffix's WHOIS hostname from the complete IANA `tlds.json` document and make one TCP port 43 query. Its `rdap_base_urls` fields do not override `dns.json`.
5. Validate the response's domain identity and extract fields. Use only embedded entities and objects already present in the same response. Never fetch registrar referrals, entity links, contact URLs, or unresolved handles.
6. Sanitize remote values and write the standardized layout. Missing and redacted values are omitted. A service error never becomes an availability claim.

The dedicated education-registry interface and DNS fallback are described in the README's [privacy policy](../README.md#privacy-policy). DNS delegation alone does not establish registration status.

## Domain and URL handling

Domain validation accepts a trailing dot and enforces DNS label and length rules. IP literals, embedded commands, URL components in a domain identity, and control characters are rejected. Unicode input uses IDNA lookup mapping. Joined emoji compatibility is limited to complete validated pictographic grapheme clusters; conversion does not establish that the registry permits registration.

Direct domain input keeps its normalized identity. URL input uses the ICANN portion of the bundled public suffix data to find the registry-level domain. Private hosting suffixes do not become registry boundaries: `https://user.github.io/` queries `github.io`. A multi-label namespace such as `net.cn` may itself be queried.

The first output row contains the ASCII domain. When a safe round-trip Unicode representation exists, an `IDN` / `国际化域名` row follows. Homoglyphs can still resemble other names; the Unicode display does not replace the ASCII identity.

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

Dates retain source precision: `YYYY-MM-DD` for dates and `YYYY-MM-DD HH:MM:SS` for timestamps. Explicitly zoned timestamps are converted to UTC and gain ` UTC`; unzoned timestamps retain their wall time. Approximate or unrecognized date text is preserved rather than assigned a fabricated precision or timezone.

RDAP `404` means no registry record, not confirmed availability. WHOIS results such as not registered, reserved, not allowed, or restricted require an explicit supported registry statement. Access denial, rate limits, timeouts, malformed data, and unknown syntax remain operational errors.

## Presentation and commands

Label width is measured in terminal display cells. The content column follows the longest label's colon by three spaces. `DNSSEC` does not determine the Chinese column. Continuations begin at the same content column; wrapping preserves grapheme clusters. Terminal dimensions take precedence over `COLUMNS`, with an 80-column fallback.

`en`, `zh`, and `auto` accept zero, one, or two leading hyphens. A standalone language command saves and confirms the default. With a domain, URL, help, version, or update, the selection applies only to that invocation and bypasses reading the saved language. The last selector wins. Device language supplies the initial default; explicit locale overrides and fallback behavior are documented in [Usage](../README.md#usage).

## Persistent state

| macOS path | Content |
| --- | --- |
| `~/Library/Caches/domain-check/dns.json` | Complete IANA RDAP bootstrap document. |
| `~/Library/Caches/domain-check/tlds.json` | Complete IANA TLD registry document. |
| `~/Library/Application Support/domain-check/config.json` | Language preference only. |

`DOMAIN_CACHE_DIR` and `DOMAIN_CONFIG_DIR` override the respective directories. No queried domains, pasted URLs, responses, or lookup history are written by the CLI.

Cache age is checked only when a lookup needs the document. One calendar month after the last successful fetch, that lookup attempts a refresh, clamping month-end dates. A validated stale document may be reused with a warning after a failed automatic refresh; its age is not reset. Invalid documents are never routing evidence.

`domain update` validates both complete downloads before replacing either file. Replacement is atomic per file, not a two-file transaction: a second save failure leaves the first file refreshed and reports failure. There are no background updates or per-suffix cache files.

## Code organization

| Package / path | Responsibility |
| --- | --- |
| [`cmd/domain`](../cmd/domain) | Executable entry point. |
| [`internal/cli`](../internal/cli) | Arguments, locale, preferences, exit status, and terminal width. |
| [`internal/registry`](../internal/registry) | Input normalization, IANA discovery, registry transports, and response parsing. |
| [`internal/output`](../internal/output) | Aligned bilingual records and grapheme-aware wrapping. |
| [`internal/safeio`](../internal/safeio) | Bounded reads of regular local data files. |
| [`scripts`](../scripts) | Installation, packaging, and installer security checks. |

Security limits and trust boundaries are maintained in [security.md](security.md). Coverage and service availability are described in [domains.md](domains.md).

## Build and release

Use Go 1.27.3 or newer. `go.mod` sets the minimum toolchain, and `go.sum` records dependency checksums. GitHub workflows use that version and run tests and vulnerability scanning before publishing. The CLI uses Go's standard network and JSON packages with bounded readers; specialized dependencies cover IDNA/public suffixes, selected legacy encodings, terminal sizing, and Unicode grapheme layout.

`make build` creates `build/bin/domain`. `make dist` creates CGO-free macOS Apple Silicon and Intel archives using `-trimpath` and stripped symbols, plus SHA-256 checksums. Each archive contains the executable, README, changelog, MIT License, version file, and this documentation directory. The standalone installer verifies the matching archive before replacing the executable. Only that explicitly invoked installer configures a standard zsh PATH entry; the CLI and `make install` install no shell behavior.

`VERSION` supplies the embedded calendar version `YY.M.D.NUM`. Release commits contain exactly `Release YY.M.D.NUM`; the matching `vYY.M.D.NUM` tag publishes release assets with an empty description. See [quality-assurance.md](quality-assurance.md#required-checks) for development checks.
