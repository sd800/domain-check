# Security policy

## Reporting a vulnerability

Please use [GitHub's private vulnerability reporting form](https://github.com/sd800/domain-check/security/advisories/new) when available. Otherwise, request a private contact from [the maintainer](https://github.com/sd800) before sharing sensitive details. Do not publish exploit details, credentials, or real registrant information in a public issue.

Include the affected release, macOS version, expected and observed behavior, and a minimal synthetic example. The latest release receives security fixes; install it before checking whether an issue remains reproducible.

## Trust boundaries and controls

- **Input is data.** Domain identities are converted to validated ASCII DNS labels. URL credentials, ports, paths, queries, and fragments are discarded locally. No user input or registry response is interpreted as shell code. The DNS fallback invokes `/usr/bin/dig` with separate arguments, without a shell or PATH lookup.
- **IANA controls routing.** Only validated service entries from the two complete IANA datasets select RDAP and WHOIS destinations. Service URLs cannot contain credentials, ports, query strings, encoded path separators, or traversal segments. WHOIS queries contain one normalized domain and, where required, fixed registry-specific selectors. Registry responses cannot supply additional destinations.
- **Connections stay at the registry.** RDAP redirects must preserve the queried domain at an IANA-listed endpoint, without HTTPS downgrade. Registrar referrals, linked contacts, and entity handles are never fetched. IANA downloads and the education-registry HTML interface do not follow redirects. Direct connections resolve once, reject non-public IP addresses, and dial the validated addresses. The CLI does not use HTTP proxy environment variables.
- **Remote data is bounded.** Response bodies are limited to 8 MiB; HTTP headers, connection times, JSON nesting and string sizes, WHOIS line counts, and extracted field counts have additional limits. Excessive records fail rather than produce silently truncated registration results. DNS retains its normal retries under the command's overall cancellation deadline.
- **Terminal output is inert.** Remote values and diagnostic details have terminal escape sequences, control characters, and Unicode formatting controls neutralized. The optional IDN display is generated locally and round-trip validated; joined emoji are the narrowly permitted formatting exception. Displayed Unicode names can still resemble other names, so the ASCII domain is always shown first.
- **Local data stays limited.** Only the language preference and two full IANA datasets are persisted. Reads accept bounded regular files, reject symlinks and files writable by other users, and do not block on named pipes. Writes use private temporary files and same-directory replacement. Lookups and registry responses are never stored by the CLI.
- **Installation is verified before replacement.** The installer uses system tools, HTTPS-only downloads and redirects, a restricted release-tag format, exact SHA-256 manifest matching, and a restricted archive allowlist. It rejects duplicate members, executable links, and traversal paths, then replaces the executable through a same-directory rename. Checksum verification detects corruption and mismatched files; it does not provide an independent signature if the release account itself is compromised.
- **Build checks are repeatable.** Go dependencies are versioned and verified against `go.sum`. GitHub Actions dependencies are pinned to full commit hashes, checkout credentials are not persisted, and check workflows have read-only repository permissions. Checks and releases run the official Go vulnerability scanner and offline installer attack tests.

## Scope and limitations

The tool trusts the local macOS account, operating system, resolver configuration, IANA, and the registry operators. Editing cached routing data or replacing files as the same user is outside this protection boundary. Syntax checks cannot establish that a compromised IANA source or registry is honest.

HTTPS uses the system's certificate validation. WHOIS has no encryption or server authentication, and an IANA-listed HTTP-only RDAP service has the same transport limitation. Network operators may observe or alter unencrypted responses. Successful parsing is not proof that disclosed ownership data is accurate or current. The DNS fallback reports delegation only and never infers availability from DNS absence.

Ordinary shell parsing takes place before the CLI starts. Quote pasted URLs containing shell metacharacters, as shown in the README. The program does not install shell aliases, widgets, or paste interception.

The ignored developer `test/` corpus is separate from runtime behavior and must remain untracked. Use synthetic data in committed security tests. Do not add query logging or persist payloads as part of diagnostics.

## Verification

With Go 1.27.3 or newer, Python 3, and network access for the public vulnerability database:

```sh
make check
make security
```

`make check` runs vet, race-enabled tests, and formatting checks. `make security` verifies module checksums, runs the pinned [official Go vulnerability scanner](https://go.dev/doc/security/vuln/), and exercises installer failure cases without downloading releases or changing the user's installation. A clean scan covers known, reachable vulnerabilities reported at scan time; it is not a guarantee that no vulnerabilities exist.
