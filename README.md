# Domain Check from Registries

A macOS command-line tool for looking up domain registration information directly from registries.

```sh
domain example.com
```

`domain` discovers registry RDAP services through [IANA's DNS bootstrap data](https://data.iana.org/rdap/dns.json). When a suffix has no RDAP entry, it discovers the registry's WHOIS server through IANA and queries that server on TCP port 43. **Registrar referrals are never followed.**

## Install

### macOS binary

Download the archive for your Mac from [Releases](https://github.com/sd800/domain-check/releases):

| Mac | Archive suffix |
| --- | --- |
| Apple Silicon | `darwin_arm64.tar.gz` |
| Intel | `darwin_amd64.tar.gz` |

Extract the archive, then install the `domain` executable into a directory on your `PATH`:

```sh
mkdir -p "$HOME/.local/bin"
install -m 755 domain "$HOME/.local/bin/domain"
```

If needed, add this line to `~/.zshrc`, then open a new terminal:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Each release includes a SHA-256 checksum file. To verify a downloaded archive, place the checksum file beside it and run `shasum -a 256` on the archive; compare its hash with the corresponding entry.

### Build from source

Requires Go 1.26 or newer.

```sh
git clone https://github.com/sd800/domain-check.git
cd domain-check
make install
```

`make install` defaults to `~/.local/bin/domain`. Use `make install PREFIX=/usr/local` to choose another prefix, or `make build` to create `bin/domain` without installing it.

With Go already installed, the command can also be installed into `$(go env GOPATH)/bin`:

```sh
go install github.com/sd800/domain-check/cmd/domain@latest
```

## Usage

```sh
domain                           # Version and a short introduction; exit code 0
domain example.com               # Saved language, or device language initially
domain -en example.com           # English
domain zh example.com            # Simplified Chinese
domain --zh example.com          # Simplified Chinese
domain example.com en            # Selectors also work after the domain
domain zh                        # Save Simplified Chinese as the default
domain en                        # Save English as the default
domain auto                      # Restore device-language detection
domain 例子.中国                  # Internationalized domain name
domain --help
domain --version
```

Running `domain` without a domain prints the program name, version, copyright notice, a short introduction, and usage examples in the default language, then exits successfully. Use `domain en` or `domain zh` to switch and save the default language. This is an offline installation smoke test and does not update the IANA cache. To also verify network access, run `domain example.com`.

Language selectors accept zero, one, or two leading hyphens:

| Language | Selectors |
| --- | --- |
| English | `en`, `-en`, `--en` |
| Simplified Chinese | `zh`, `-zh`, `--zh` |
| Device language | `auto`, `-auto`, `--auto` |

The last language selector wins and is saved for future invocations, including when supplied alongside a domain. With no saved selection, or after `domain auto`, the command uses the Mac's first preferred language, with Chinese mapped to Simplified Chinese and other languages mapped to English. In automatic mode, explicit `LC_ALL` or `LC_MESSAGES` values override the device preference; `LANG` is the fallback when the preference cannot be read.

Preferences are stored at `~/Library/Application Support/domain-check/config.json` on macOS. Set `DOMAIN_CONFIG_DIR` to use another configuration directory. Preferences are separate from the IANA cache, so deleting cached bootstrap data does not reset the language. Invalid arguments do not change the saved preference. A configuration read or write failure produces a warning; the current command can still run.

Pass one domain per invocation, without a URL scheme, port, or path. A trailing dot is accepted. Internationalized names are converted to their ASCII/Punycode form for querying and display. The exact domain supplied is queried; subdomains are not silently replaced with a parent domain.

## Output

An illustrative registry response:

```text
Domain Name:         example.com

Registration Time:   2003-03-10 11:19:18
Expiration Time:     2027-05-15 11:25:25
Last Changed Time:   2025-05-15 10:25:25

Registrar:           Example Registrar
Domain Status:       ok

Registrant:          Example Organization
Registrant Email:    admin@example.com

Name Server:         ns1.example.com
Name Server:         ns2.example.com
DNSSEC:              unsigned
```

The labels in Chinese are:

```text
域名:         example.com

注册时间:     2003-03-10 11:19:18
过期时间:     2027-05-15 11:25:25
更新时间:     2025-05-15 10:25:25

注册商:       Example Registrar
状态:         ok

注册人:       Example Organization
联系邮箱:     admin@example.com

域名服务器:   ns1.example.com
域名服务器:   ns2.example.com
DNSSEC:       unsigned
```

Only fields supplied by the registry are shown. Missing, empty, or explicitly redacted values are omitted. A registry may disclose no registrant name or email; the command does not contact a registrar to fill those gaps. Registry status values and names retain their source language.

Content begins in a fixed column determined by the longest label, with three spaces after that label's colon. Chinese widths use terminal display cells; `DNSSEC` does not determine the Chinese column. Long values wrap to the terminal width, with continuation lines starting in the content column. When terminal dimensions are unavailable, `COLUMNS` is used, falling back to 80 columns. Very narrow terminals must still accommodate the label column and at least one character.

Timestamps with explicit UTC offsets are converted to UTC and rendered as `YYYY-MM-DD HH:MM:SS`. Unzoned WHOIS timestamps and date-only values retain their original representation. Dates are displayed even when the registry supplies no time of day. The RDAP database refresh timestamp is not substituted for a domain's last-changed time. DNSSEC is shown only when the registry supplies enough information to determine it.

### Missing records and registration restrictions

```text
Domain Name:         example.com
Result:              No registry record
```

```text
域名:         example.com
查询结果:     注册局无记录
```

A normal RDAP `404` means the registry returned no matching record. It does **not** establish that the domain can be registered. `Not registered`, `Reserved`, or `Registration not allowed` is shown only when a supported WHOIS response explicitly states that result. Timeouts, access denials, rate limits, unsupported suffixes, and malformed responses are reported as errors on stderr.

| Exit code | Meaning |
| --- | --- |
| `0` | A registry record was returned, or the introduction, help, or version was displayed |
| `1` | No registry record, or an explicit registration restriction |
| `2` | Invalid arguments or domain |
| `3` | Discovery, network, service, response, or output error |

## Registry-only queries

- RDAP requests use endpoints listed by IANA for the domain's matching suffix, preferring HTTPS.
- Response links and entity links are never fetched, including registrar `related` links.
- HTTP redirects are accepted only for the same domain query at an IANA-listed endpoint for that suffix, including an HTTP-to-HTTPS upgrade. Arbitrary paths, external hosts, and redirects with query parameters are blocked.
- WHOIS discovery queries `whois.iana.org` for the suffix, then queries only the registry host in IANA's `whois` field. Registrar WHOIS fields and referrals in the registry response are ignored.
- When IANA lists RDAP, an RDAP failure does not trigger a WHOIS fallback.

WHOIS response formats vary by registry. The parser handles common labeled formats and multiline sections, including CNNIC and Nominet-style records, but cannot normalize every registry-specific format. Unrecognized replies produce errors instead of guessed registration results. Outbound TCP port 43 must be available to use WHOIS.

## IANA cache

On macOS, IANA's `dns.json` is cached at:

```text
~/Library/Caches/domain-check/dns.json
```

Discovered registry WHOIS hosts are cached under `whois/` in the same directory. Set `DOMAIN_CACHE_DIR` to use another cache directory.

Cache freshness is checked only during a domain lookup. After one calendar month from the last successful fetch, the next lookup refreshes the data. Month-end dates are clamped: a January 31 fetch becomes due on February 28, or February 29 in a leap year. The IANA publication date does not determine local cache age. Help and version commands do not perform network requests.

If refresh fails, a previously validated cache is used with a warning; a later lookup retries the refresh. Invalid caches are never used. Successful downloads replace cache files atomically. No background process, scheduled job, or lookup-result cache is created.

## Development and releases

```sh
make check       # Static checks, race-enabled offline tests, formatting
make build       # Native bin/domain
make dist        # Apple Silicon and Intel archives with checksums
```

Offline tests verify registry-only routing, redirect/referral blocking, RDAP error handling, WHOIS discovery, monthly cache behavior, field extraction, language selectors, and Unicode output alignment.

`VERSION` is the single source of truth for the embedded version. Versions follow `YY.M.D.NUM`, where `NUM` starts at `1` and increments for each release on the same date. Months, days, and sequence numbers are not zero-padded. Release commits use exactly `Release YY.M.D.NUM`, without a commit body. Changes are recorded in the English [changelog](CHANGELOG.md).

Pushing a `vYY.M.D.NUM` tag runs the release workflow, checks its agreement with `VERSION` and the commit message, and publishes macOS archives and checksums with the title `Release YY.M.D.NUM` and an empty release description.

## References

- [IANA RDAP bootstrap data](https://www.iana.org/assignments/rdap-dns/)
- [IANA WHOIS service](https://www.iana.org/whois)
- [RFC 7480: RDAP over HTTP](https://www.rfc-editor.org/rfc/rfc7480)
- [RFC 9083: RDAP JSON responses](https://www.rfc-editor.org/rfc/rfc9083)
- [RDAP Web Client](https://github.com/rdap-org/client.rdap.org), a reference for direct registry discovery, HTTPS endpoint preference, and the distinction between registry responses and registrar referrals. This CLI is an independent Go implementation.

## License

[MIT License](LICENSE)
