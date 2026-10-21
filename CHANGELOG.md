# Changelog

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
