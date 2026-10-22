# Security policy

Domain Check from Registries protects domain lookups through direct registry connections, local processing, validated inputs, and verified installation. Security checks are part of every release.

## Reporting a vulnerability

Share security findings privately through [GitHub's vulnerability reporting form](https://github.com/sd800/domain-check/security/advisories/new), or coordinate a private report with [the maintainer](https://github.com/sd800). Keep sensitive technical details and personal information in the private report.

Include the affected release, macOS version, expected and observed behavior, and a minimal synthetic example. Security updates are delivered in the latest release.

## Security controls

**Validated input and command execution.** Domain identities are converted to validated ASCII DNS labels. URL credentials, ports, paths, queries, and fragments are removed locally before querying. Inputs and responses are handled as data. The DNS fallback invokes the fixed macOS system executable `/usr/bin/dig` with separate arguments, keeping command execution independent of shell interpretation and PATH lookup.

**Authoritative service discovery.** Validated entries from IANA's complete discovery datasets select RDAP and WHOIS destinations. The explicit `.tw` WHOIS route also obtains its registry host from the complete IANA dataset; response referrals never select another destination. Single-label TLD allocation queries use the fixed IANA RDAP endpoint and retain the same response and connection controls. Service URL validation rejects credentials, ports, query strings, encoded separators, and traversal segments. WHOIS queries contain one normalized domain and any fixed selector required by that registry.

**Direct registry connections.** Redirects preserve the queried domain at the selected registry endpoint and retain HTTPS protection. Destinations come from complete IANA metadata or fixed, documented namespace-registry routes for `.ac.uk` and `.gov.uk`. Registrar referrals, linked contacts, and entity handles are never fetched. IANA downloads and the education-registry HTML interface use their configured endpoints directly. Each connection validates the resolved public IP addresses and connects to those addresses. The CLI uses direct connections, independent of HTTP proxy environment variables.

**Bounded response processing.** Response bodies are limited to 8 MiB, with additional limits on HTTP headers, connection times, JSON nesting and string sizes, WHOIS line counts, and extracted fields. Responses exceeding these limits produce an explicit error. DNS retains its normal retries under the command's overall cancellation deadline.

**Safe terminal presentation.** Remote fields and diagnostics are sanitized to neutralize terminal escape sequences, control characters, and Unicode formatting controls. Internationalized domain display is generated locally and round-trip validated, including supported joined emoji. The ASCII identity is always displayed first for consistent domain identification.

**Minimal local storage.** Persistent application data consists of the CLI language preference, two complete IANA datasets, and optional local-web settings and LaunchAgent configuration. The browser stores only theme and language choices. Reads accept bounded regular files and reject symlinks, special files, and files writable by other users. Writes use private temporary files and atomic replacement. Queried domains, pasted URLs, and registry responses remain in memory for the current lookup.

**Loopback-only browser access.** The optional web service binds IPv4 and IPv6 loopback addresses only. Requests must use an allowlisted Host, preventing DNS-rebinding access through arbitrary hostnames. Browser requests are checked against their Origin and Fetch Metadata. Lookup POSTs require JSON and a cryptographically random per-process token; the service sends no cross-origin access permissions. Request bodies, request times, concurrency, and query frequency are bounded.

**Safe browser rendering.** Registration values enter the page through DOM text nodes. URLs, email addresses, and phone numbers remain plain text; browser format detection is disabled and result values are never converted to links. A restrictive Content Security Policy permits only embedded local assets and the exact startup script hash, blocks framing and base-URL overrides, and disables external connections. Responses use no-store caching, no-referrer, and MIME-sniffing protection. The server keeps no access logs or lookup-result storage. Browser controls save appearance and language preferences independently of query data.

**Current-user service management.** Setup uses the current account's LaunchAgent and fixed system executables. Paths are passed as separate arguments and escaped when written to XML. The service requires no administrator access and listens on neither LAN nor public interfaces. Repeating an enabled setup option preserves the existing registration. Executable updates reuse a loaded job without rewriting startup files or re-enabling an unloaded service. Ordinary CLI commands do not manage startup registrations. Disabling the interface unloads its job and removes startup registrations.

**Verified installation.** The installer uses macOS system tools and HTTPS-only downloads. It validates release tags, matches the downloaded archive against the published SHA-256 manifest, and checks archive members before installation. Duplicate members, executable links, and traversal paths are rejected. The executable is replaced through a same-directory rename after verification.

**Controlled builds and releases.** Go dependencies are versioned and verified against `go.sum`. GitHub Actions dependencies are pinned to full commit hashes. Checkout credentials are not persisted, and check workflows use read-only repository permissions. Checks and releases run the official Go vulnerability scanner and isolated installer security tests.

## Platform and protocols

Domain Check runs within the current macOS user's environment, using the operating system's resolver and certificate validation. IANA supplies service assignments, and registries supply registration data. HTTPS is preferred whenever a registry advertises it. WHOIS and HTTP-only RDAP use unencrypted connections. Browser-to-service HTTP stays on the Mac's loopback interface; registry connections retain the same transport and destination controls as CLI queries.

For TLD queries, IANA receives the queried TLD as the authoritative root registry; ordinary domain discovery uses complete downloads without revealing the requested suffix to IANA. Results retain the registry's disclosed information and distinguish registration records, explicit policy results, and service errors. The education-registry DNS fallback provides public nameserver records.

Quote pasted URLs containing shell metacharacters so the shell passes the complete URL to the CLI. Examples are provided in the README's [Usage section](../README.md#usage).

## Verification

With Go 1.27.3 or newer, Python 3, and network access for the public vulnerability database:

```sh
make check
make security
```

`make check` runs vet, race-enabled tests, and formatting checks. `make security` verifies module checksums, checks reachable code against the [official Go vulnerability database](https://go.dev/doc/security/vuln/), and exercises installer failure cases in an isolated environment. The [quality assurance guide](quality-assurance.md) documents the complete validation and release process.
