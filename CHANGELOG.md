# Changelog

## 26.10.23.3

- Introduce a bilingual product website with local browsing support and GitHub Pages publishing.

## 26.10.23.2

- Standardize internal application naming and macOS integration.

## 26.10.23.1

- Refine registrant information layout in Terminal and local browser results.

## 26.10.22.13

- Improve WHOIS compatibility and service-response handling across registry fallback services.

## 26.10.22.12

- Expand disclosed registrant contact details while preserving established registry-specific layouts.
- Extend registry WHOIS fallback to country-code and generic RDAP-enabled domains.

## 26.10.22.11

- Clarify registry attribution in lookup results and documentation.

## 26.10.22.10

- Refine README guidance for product usage, browser setup, and privacy.

## 26.10.22.9

- Add registry WHOIS fallback for country-code domains and expand special-namespace normalization.
- Classify .tw as WHOIS-only in routing and coverage.

## 26.10.22.8

- Show disclosed registrar country codes in ordinary-domain RDAP results.
- Improve registry routing, field normalization, and special-namespace result handling.

## 26.10.22.7

- Refine local service naming and integration.

## 26.10.22.6

- Preserve the full disclosed EDUCAUSE registrant block in aligned CLI and browser results.

## 26.10.22.5

- Refine TLD result grouping in the CLI and browser.

## 26.10.22.4

- Add direct IANA RDAP queries for TLD allocation records with aligned contact and delegation details in the CLI and browser.
- Accept one leading dot on domain queries and rename the device-language selector to auto-lang, preserving command priority.

## 26.10.22.3

- Highlight setup menu choices in cyan on terminals, keeping redirected output plain and respecting NO_COLOR and basic-terminal settings.
- Reuse existing web-service registrations during updates and repeated setup, preserving startup files and unloaded service preferences.

## 26.10.22.2

- Use Chinese year, month, and day formatting in Chinese CLI and browser results while retaining source precision, time, and UTC labels.

## 26.10.22.1

- Add an optional local browser interface with direct domain and URL lookups, bilingual results, adaptive appearance, and a text-free logo.
- Add current-user setup controls for web access and login startup, with private local operation and automatic refresh after executable updates.
- Extend security checks, browser and service validation, and public documentation for the new interface.

## 26.10.21.32

- Refine the project overview.

## 26.10.21.31

- Update the project overview.

## 26.10.21.30

- Update the project overview.

## 26.10.21.29

- Refine public project documentation for clarity and consistency.

## 26.10.21.28

- Document macOS 13 (Ventura) as the minimum supported system for executables built with the updated Go toolchain.
- Check macOS compatibility before downloading or replacing the executable, and cover unsupported and invalid system versions with isolated installer tests.

## 26.10.21.27

- Require Go 1.27.3 for local and release builds, replacing the older CI toolchain flagged by vulnerability scanning.
- Add public technical, domain-coverage, security, quality-assurance, and reference documentation with a consolidated README index.
- Move the root security policy into docs/security.md and include the public documentation in release archives.
- Group generated binaries and archives under build/, update installation and publishing paths, and keep the archive allowlist restricted to the documented layout.

## 26.10.21.26

- Harden injection boundaries with exact WHOIS query validation, system-path DNS execution, defensive diagnostic sanitization, and matching RDAP ASCII and Unicode identities.
- Validate service hostnames, suffixes, and URL paths; reject non-public destinations and pin connections to validated DNS results without environment proxies.
- Bound local cache and preference reads, reject symlinks and special files, and limit recursive response parsing and excessive record fields.
- Require HTTPS-only installer downloads, validate checksum manifests and archive members, and replace executables atomically without following destination symlinks.
- Add security regression tests, parser and input fuzzing, isolated installer attack tests, an English security policy, pinned workflow actions, and official Go vulnerability scanning for checks and releases.

## 26.10.21.25

- Audit every current IANA WHOIS-only TLD, including non-country namespaces and IDN scripts, and add the missing registry-family syntax rules.
- Extract Lithuanian registrant contacts, Luxembourg domain status, Turkmen organization names, Ugandan update dates, and Ukrainian IDN identities and fields.
- Normalize dates for additional IDN registry families while preserving source precision, contact boundaries, nameserver records, and DNSSEC data.
- Recognize explicit registry reservation and policy notices, distinguish query-syntax and service failures from availability, and validate domain names in negative replies.
- Add synthetic regression coverage for the observed formats, contact-email isolation, notice precedence, invalid queries, and contradictory identities.

## 26.10.21.24

- Add the privacy policy as the final README section, covering local processing, stored data, registry queries, independent services, and the .edu.cn DNS fallback.

## 26.10.21.23

- Remove automatic legacy WHOIS-cache cleanup and migration diagnostics; keep runtime behavior focused on the two complete IANA datasets.

## 26.10.21.22

- Replace per-suffix WHOIS discovery caches with IANA's complete TLD registry JSON, keeping cached contents independent of queried suffixes.
- Remove the legacy WHOIS cache on the next lookup or explicit update, and query registry WHOIS servers directly without a separate IANA port-43 discovery request.
- Refresh both complete IANA datasets with `domain update`, retain monthly validation and stale-cache handling, and preserve RDAP routing precedence.
- Share bounded metadata downloads and cache loading, reject invalid or inactive TLD routing entries, and cover privacy, migration, IDN routing, refresh failures, and redirect blocking with offline tests.

## 26.10.21.21

- Replace the long, repeated README installation blocks with a short copy-and-paste entry point and a commented shared installer.
- Keep checksum verification, temporary-file cleanup, and repeatable PATH setup; reuse the installer for optional source builds.
- Remove redundant Go and manual-install command blocks, link release downloads, and fold source installation details away from the main instructions.

## 26.10.21.20

- Put a complete macOS Terminal installation command first, with automatic architecture selection, latest-release download, and SHA-256 verification without requiring Go.
- Combine installation and repeatable PATH setup in every README method so the command works in the current terminal and future zsh sessions.
- Use the same executable directory for Go, source, and binary installations, and show the installed version after each successful setup.

## 26.10.21.19

- Remove the zsh integration, URL paste widget, shell aliases, and shell-configuration installer; restore installation and removal of the CLI executable only.
- Package only the executable and documentation, and document normal shell quoting for URLs while retaining temporary language selection and emoji domain support.

## 26.10.21.18

- Keep `en`, `zh`, and `auto` selectors temporary when combined with a domain, URL, help, version, or update; save the default only for standalone language commands.
- Avoid writing language preferences during queries, including when the configuration directory is unavailable, while preserving standalone confirmation and failure reporting.
- Update English and Simplified Chinese help, the README, and regression coverage to distinguish temporary language selection from saved defaults.
- Add command-specific zsh glob protection and automatic quoting for pasted URLs, preserving query parameters, shell separators, substitutions, Unicode, and emoji as literal argument data.
- Include a repeatable installer and shell integration in macOS archives, preserve existing zsh settings and symlinks, and keep staged installations separate from user shell configuration.
- Support joined emoji labels through constrained IDNA compatibility handling, preserve their identity during WHOIS parsing and Unicode output, and keep grapheme clusters intact when wrapping.

## 26.10.21.17

- Confirm successful standalone `en`, `zh`, and `auto` language changes after saving the preference, including all supported hyphen spellings.
- Show explicit language confirmations in the selected language and automatic-mode confirmations in the device language, while keeping the no-argument introduction and query output intact.
- Report standalone preference-save failures with exit code `3` instead of displaying a successful introduction; queries can still continue with a warning.

## 26.10.21.16

- Show the locally decoded Unicode name directly below an internationalized domain's ASCII name, labeled `IDN` in English or `国际化域名` in Chinese, when IDNA conversion and identity validation succeed.
- Apply the existing content alignment and wrapping to the new row, including missing-record and registration-restriction results.
- Rename the README output introduction to describe the standardized query-result layout while keeping its examples unchanged.

## 26.10.21.15

- Revise the README introduction to highlight readable registration details and show both domain-name and full-URL queries.

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
