# Changelog

## 26.10.21.14

- Add the product statement as the first README section, describing direct registry queries and consistent, readable output.

## 26.10.21.13

- Reduce executable size by replacing general HTML encoding discovery with the registry interface's required GBK, GB2312, GB18030, and UTF encoding handling.
- Extract preformatted registry text with a streaming HTML tokenizer instead of constructing a complete document tree, preserving entities, nested text, line breaks, and script/style exclusion.
- Tokenize WHOIS labels and values once for shared domain, contact, and included-object processing; collapse redundant blank separators and avoid a separate object-line index map.
- Reduce temporary allocations when sanitizing already normalized text, and replace simple label regular expressions with direct checks.
- Centralize observed registry date and contact-format differences in compact profiles while preserving response identity, redaction, date precision, and registry-only routing.
- Recognize EDUCAUSE missing-domain replies as absent registry records, preserve registrant/contact boundaries, and parse its month-name dates directly without repeated generic format attempts.
- Preserve IANA infrastructure-domain metadata without requiring nameservers, and recognize zero-object WHOIS replies without implying registration availability.
- Support explicitly declared Chinese IDN bundles, compact Hong Kong suffixes, Taiwan character-variant tables, scoped registrant emails, and UTC offsets in registry timestamps.
- Normalize IANA's Unicode WHOIS suffix identities before comparing them with ASCII IDNA queries and caching the verified registry endpoint.
- Match Unicode names in negative replies, distinguish unavailable names from permanent reservation, and hide bracketed or parenthesized redaction placeholders.
- Add parser benchmarks and regression coverage for encoding selection, HTML extraction boundaries, DNS fallback, and terminal-text sanitization.

## 26.10.21.12

- Separate domain managers from explicit registrant sections, including slash-separated owner headings, and keep redacted owners from being replaced by management contacts.
- Recover domain dates after empty WHOIS referral fields and domain-details headings, and accept complete registry records that omit nameservers.
- Recognize additional registry absence, availability, reservation, and policy-rejection replies, including status fields and protocol comments, while validating named results against the requested domain.
- Keep copyright and terms-of-use text from changing lookup results, and preserve errors for access denials, unsupported queries, and incomplete responses.
- Handle restricted CNNIC namespace and domain queries with a distinct online-registration restriction result, without following registrar suggestions or assuming permanent reservation.
- Accept pasted URLs whose hostname is a multi-label namespace such as `net.cn`, preserving the exact namespace for registry queries.
- Expand synthetic regression coverage for exceptional registry-owned records, ordinary registrations, negative replies, response identity, and registry-only routing.
- Remove older generated release archives when packaging so local builds retain only the current release artifacts.

## 26.10.21.11

- Add `domain update`, `domain -update`, and `domain --update` to refresh IANA DNS bootstrap data immediately, regardless of cache age.
- Validate downloads before atomically replacing the cache, report explicit update failures, and preserve the existing file when a refresh fails.
- Support zero, one, or two leading hyphens consistently for update, help, and version commands, with localized output and language selectors.
- Fix standalone and empty WHOIS contact labels being mistaken for registrant values, including `Address` in `.im` responses.
- Prevent a later manager's details from replacing a redacted owner, recover domain expiry dates after contact sections, and normalize `.im` day-first timestamps.
- Expand regression tests for forced updates, failure preservation, contact boundaries, redacted values, and dates supplied on separate lines.

## 26.10.21.10

- Broaden WHOIS field matching across registry formats for ASCII top-level domains without RDAP, including dot leaders, bracketed labels, colonless dates, bundled domain names, and multiline records.
- Resolve registrant, nameserver-set, and DNSSEC-key objects included in the same registry response without making additional queries or following registrar referrals.
- Keep domain dates and registrant details separate from administrative, technical, and registrar contact objects; hide unresolved contact handles and redacted data.
- Normalize additional registry-specific date formats and explicit timezones while preserving approximate dates and unknown timezone abbreviations.
- Add registry-specific query syntax and legacy character decoding, and prevent WHOIS footer text from becoming a false availability result.
- Expand synthetic regression coverage for registry formats, contact scoping, response identity, DNSSEC, referral prevention, and lookup privacy.

## 26.10.21.9

- Normalize date-only values to `YYYY-MM-DD` and timestamps to `YYYY-MM-DD HH:MM:SS`, preserving the supplied precision.
- Add a `UTC` suffix to timestamps converted from explicitly zoned values; leave unzoned values unlabeled.
- Accept pasted URLs, strip URL components, and query the registrable domain using public-suffix rules while keeping direct domain inputs exact.
- Keep URL input handling separate from strict registry-response validation, and never fetch the pasted URL itself.
- Verify that lookups do not persist query inputs or results; keep local application data limited to service metadata and language preferences.

## 26.10.21.8

- Recognize EDUCAUSE WHOIS activation, expiration, and last-update fields so `.edu` registration dates are displayed.
- Verify that date-only WHOIS values and RDAP events retain their original date precision and appear in the output.
- Remove domain-specific tuning details from the README.

## 26.10.21.7

- Apply the three-second deadline only to `.edu.cn` registry HTTP requests.
- Use dig's default timeout and retry settings for DNS fallback, preserving overall command cancellation.

## 26.10.21.6

- Query `.edu.cn` domains directly through the dedicated NIC registry interface with a three-second HTTP deadline.
- Decode GBK and GB2312 HTML before extracting the preformatted WHOIS record, organization name, and domain servers.
- Immediately fall back to bounded `dig NS` queries on access denial or other registry failures, displaying only the domain and returned nameservers.
- Keep DNS failures separate from registration-availability results and prevent HTML links or redirects from triggering additional queries.

## 26.10.21.5

- Persist explicit English and Simplified Chinese selections as the default for subsequent invocations.
- Add `auto`, `-auto`, and `--auto` to restore device-language detection.
- Store language preferences separately from the IANA cache, with atomic writes and warnings for configuration failures.

## 26.10.21.4

- Display the program name, version, and Songming.org copyright notice on separate lines in the introduction.
- Use `domain zh example.com` consistently in usage examples.
- Compact registration output by grouping the registrar with domain statuses and nameservers with DNSSEC.

## 26.10.21.3

- Show a localized introduction, version, and usage examples when `domain` is run without a domain, including with a language selector alone.
- Return exit code `0` for this offline installation check without accessing registry services or updating the IANA cache.

## 26.10.21.2

- Update the MIT License attribution to Songming.org.
- Use the full MIT License name in the README.

## 26.10.21.1

- Add the `domain` command for direct registry domain-registration lookups.
- Discover registry RDAP services through IANA's DNS bootstrap data and query registry WHOIS on TCP port 43 only when RDAP is absent.
- Block registrar referrals, entity-link fetching, and redirects outside the IANA-listed registry endpoints.
- Cache IANA data locally and refresh it during lookups after one calendar month; retain validated stale data when refresh fails.
- Detect the macOS device language and support English and Simplified Chinese output through `en`, `-en`, `--en`, `zh`, `-zh`, and `--zh` selectors.
- Align fields using Unicode display widths, indent wrapped content, and omit unavailable or redacted fields.
- Convert zoned timestamps to UTC while preserving unzoned WHOIS timestamps.
- Distinguish missing registry records from explicit unregistered, reserved, or registration-prohibited responses.
- Add internationalized domain-name support, bounded network requests, offline protocol tests, and macOS builds for Apple Silicon and Intel.
- Embed the calendar version from the root `VERSION` file and add automated checks and release packaging.
