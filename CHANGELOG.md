# Changelog

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
