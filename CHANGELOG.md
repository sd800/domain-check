# Changelog

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
