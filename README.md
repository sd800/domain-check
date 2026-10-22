# Domain Check from Registries

A macOS command-line tool that looks up domain registration details directly from registries and returns them in a clean, readable layout.

```sh
domain example.com
domain https://www.example.com/path
```

Type a domain name or paste a full URL straight from the browser. The command picks out the domain and prints its key registration details in one aligned view.

## Product statement

Domain Check from Registries is built around a simple principle: **a domain lookup should go straight to the source and come back easy to read.** It works right in the macOS Terminal, from query out to answer back.

**Send the query straight to the registry.** Searching through commercial domain websites or following registrar referrals exposes interest in a domain name to tracking, giving others a signal to front-run it or hold onto it. Domain Check queries the registry's own RDAP and WHOIS services and never follows registrar referrals. Whether the input is a domain name or a full URL, every lookup goes directly to the registry and nowhere else.

**Bring back a clean, readable answer.** Registry responses often contain inconsistent formatting, redundant fields, and technical details that obscure useful information. Domain Check extracts relevant registration details, organizes them into a consistent, easy-to-scan format, and drops the clutter. Whether the response is RDAP or registry-specific WHOIS, every answer comes back in the same clean layout.

**One command, one clean answer, straight from the registry.**

## Install

Copy this entire block into macOS Terminal. It installs the latest release for Apple Silicon or Intel, verifies its SHA-256 checksum, and makes `domain` available immediately. No Go installation or manual download is required.

```sh
/bin/sh <<'SH' && export PATH="$HOME/.local/bin:$PATH" && domain --version
set -eu
installer=$(mktemp)
trap 'rm -f "$installer"' EXIT
curl -fsSL --retry 3 -o "$installer" \
  https://raw.githubusercontent.com/sd800/domain-check/main/scripts/install.sh
sh "$installer"
SH
```

The [commented installer](scripts/install.sh) puts the executable in `~/.local/bin` and adds a standard `PATH` entry to `${ZDOTDIR:-$HOME}/.zshrc` only once. The block above also sets `PATH` in the current terminal and prints the installed version; no restart is needed. Run the same block again to install the latest release.

Archives and checksum files are also available on the [Releases page](https://github.com/sd800/domain-check/releases).

<details>
<summary>Build and install from source</summary>

Requires Go 1.26 or newer and Git. Run this block from the directory where you want the checkout. It builds the executable, uses the same installer for installation and future zsh sessions, then enables `domain` in the current terminal:

```sh
git clone https://github.com/sd800/domain-check.git &&
cd domain-check &&
make build &&
sh scripts/install.sh bin/domain &&
export PATH="$HOME/.local/bin:$PATH" &&
domain --version
```

To build without installing, run `make build` and use `./bin/domain`.

</details>

## Usage

```sh
domain                           # Version and a short introduction; exit code 0
domain example.com               # Saved language, or device language initially
domain -en example.com           # English for this query
domain zh example.com            # Simplified Chinese for this query
domain --zh example.com          # Simplified Chinese for this query
domain auto example.com          # Device language for this query
domain example.com en            # Selectors also work after the domain
domain zh                        # Save Simplified Chinese as the default
domain en                        # Save English as the default
domain auto                      # Restore device-language detection
domain update                    # Refresh IANA RDAP and WHOIS data immediately
domain --update                  # Same explicit refresh
domain 例子.中国                  # Internationalized domain name
domain 'https://www.example.com/path?item=1&lang=zh'
domain --help
domain --version
```

Running `domain` without arguments prints the program name, version, copyright notice, a short introduction, and usage examples in the default language, then exits successfully. This is an offline installation smoke test and does not update the IANA cache. To also verify network access, run `domain example.com`.

Use `domain en` or `domain zh` to switch and save the default language, or `domain auto` to restore device-language detection. These standalone language commands print a short confirmation after the preference has been saved, without performing a lookup or updating the IANA cache.

Language selectors accept zero, one, or two leading hyphens:

| Language | Selectors |
| --- | --- |
| English | `en`, `-en`, `--en` |
| Simplified Chinese | `zh`, `-zh`, `--zh` |
| Device language | `auto`, `-auto`, `--auto` |

The `update`, `help`, and `version` commands also accept zero, one, or two leading hyphens. For example, `domain update`, `domain -update`, and `domain --update` perform the same refresh. `-h` and `-v` remain available as shortcuts for help and version. Language selectors work with these commands, such as `domain zh update`. Run `update` without a domain or URL.

The last language selector wins for the current invocation. When supplied with a domain, URL, `help`, `version`, or `update`, it applies only to that invocation and does not change the saved default. Only standalone `domain en`, `domain zh`, or `domain auto` commands save the default language. With no saved selection, or after standalone `domain auto`, the command uses the Mac's first preferred language, with Chinese mapped to Simplified Chinese and other languages mapped to English. In automatic mode, explicit `LC_ALL` or `LC_MESSAGES` values override the device preference; `LANG` is the fallback when the preference cannot be read.

Preferences are stored at `~/Library/Application Support/domain-check/config.json` on macOS. Set `DOMAIN_CONFIG_DIR` to use another configuration directory. Preferences are separate from the IANA cache, so deleting cached bootstrap data does not reset the language. Invalid arguments do not change the saved preference. If a standalone language command cannot save its preference, it reports the failure and returns exit code `3`. If reading a saved preference fails, other commands can continue in the device language with a warning. An explicit language selector bypasses reading the saved preference.

Pass one domain or URL per invocation. URLs are automatically reduced to their registrable domain: `https://www.example.com:8443/path?item=1` queries `example.com`, and `https://www.example.co.uk/` queries `example.co.uk`. Addresses without a scheme, such as `www.example.com/path`, are also accepted. The URL's scheme, credentials, port, path, query, and fragment are not sent to the registry, and the URL itself is never fetched. Private hosting suffixes are reduced to the registry-level domain; for example, `https://user.github.io/` queries `github.io`.

Quote URLs containing shell-special characters such as `?`, `&`, `*`, or parentheses, as shown above, so the shell passes the complete URL to `domain`.

Direct domain inputs are queried exactly as supplied. A trailing dot is accepted. Internationalized names, including emoji and joined emoji sequences, are converted to ASCII/Punycode for querying. The domain line shows that ASCII name, with a separate Unicode display name when it can be decoded and validated.

## Output

Standardized layout for query results:

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

The standardized layout with Simplified Chinese labels:

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

WHOIS parsing supports registry-specific labels, contact sections, linked contact objects included in the same response, and multiline nameserver and DNSSEC records. Domain dates remain separate from contact-object dates, and administrative, technical, and registrar email addresses are not substituted for a registrant email. Coverage depends on the registry's public service and disclosed fields; access restrictions, timeouts, and unrecognized responses do not imply that a domain is available.

Content begins in a fixed column determined by the longest label, with three spaces after that label's colon. Chinese widths use terminal display cells; `DNSSEC` does not determine the Chinese column. Long values wrap to the terminal width, with continuation lines starting in the content column. When terminal dimensions are unavailable, `COLUMNS` is used, falling back to 80 columns. Very narrow terminals must still accommodate the label column and at least one character.

Date-only values use `YYYY-MM-DD`, including values returned as `23-May-1985`. Timestamps use `YYYY-MM-DD HH:MM:SS`. Explicitly zoned timestamps are converted to UTC and labeled with a ` UTC` suffix. Unzoned timestamps retain their source wall time without a timezone label; date-only records retain date precision. Approximate or unrecognized date text is preserved. The RDAP database refresh timestamp is not substituted for a domain's last-changed time. DNSSEC is shown only when the registry supplies enough information to determine it.

### Missing records and registration restrictions

```text
Domain Name:         example.com
Result:              No registry record
```

```text
域名:         example.com
查询结果:     注册局无记录
```

A normal RDAP `404` means the registry returned no matching record. It does **not** establish that the domain can be registered. `Not registered`, `Reserved`, or `Registration not allowed` is shown only when a supported WHOIS response explicitly states that result. `Not available for registration` preserves an explicit registry availability restriction without assuming the reason. A reply that only rules out online registration is shown as `Registration restricted (online registration unavailable)`; it does not establish permanent reservation. Timeouts, access denials, rate limits, unsupported suffixes, and malformed responses are reported as errors on stderr.

| Exit code | Meaning |
| --- | --- |
| `0` | A registry record was returned, language preferences or IANA data were saved, or the introduction, help, or version was displayed |
| `1` | No registry record, or an explicit registration restriction |
| `2` | Invalid arguments or domain |
| `3` | Discovery, configuration, network, service, response, or output error |

## Registry-only queries

- RDAP requests use endpoints listed by IANA for the domain's matching suffix, preferring HTTPS.
- Response links and entity links are never fetched, including registrar `related` links.
- HTTP redirects are accepted only for the same domain query at an IANA-listed endpoint for that suffix, including an HTTP-to-HTTPS upgrade. Arbitrary paths, external hosts, and redirects with query parameters are blocked.
- When RDAP is absent, WHOIS uses the registry host in IANA's complete `tlds.json` dataset, then queries that registry directly on TCP port 43. Registrar WHOIS fields and referrals in the registry response are ignored. The `rdap_base_urls` field in `tlds.json` does not override `dns.json`.
- When IANA lists RDAP, an RDAP failure does not trigger a WHOIS fallback.

WHOIS response formats vary by registry. The parser handles common labeled formats and multiline sections, including CNNIC and Nominet-style records, but cannot normalize every registry-specific format. Unrecognized replies produce errors instead of guessed registration results. Outbound TCP port 43 must be available to use WHOIS.

## IANA cache

The tool caches two complete, official IANA datasets on macOS:

```text
~/Library/Caches/domain-check/dns.json
~/Library/Caches/domain-check/tlds.json
```

`dns.json` determines RDAP routing. The complete TLD registry in `tlds.json` supplies WHOIS server addresses; entries are not selected or written according to queried suffixes. Set `DOMAIN_CACHE_DIR` to use another cache directory.

Cache freshness is checked only during a lookup, when the corresponding dataset is needed. After one calendar month from its last successful fetch, the next lookup that needs that file refreshes it. Month-end dates are clamped: a January 31 fetch becomes due on February 28, or February 29 in a leap year. The IANA publication date does not determine local cache age. The introduction, help, version, and standalone language commands do not fetch or refresh IANA data.

Run `domain update` to download and validate both current datasets immediately, even when the local caches are fresh. Both downloads are validated before either cached file is replaced. Each replacement is atomic and restarts that file's one-month freshness period. Download, validation, or cache-write failures return exit code `3`; an incomplete update never reports success. If saving one file succeeds before saving the other fails, that file remains refreshed and the other keeps its previous contents.

If an automatic refresh fails, a previously validated cache is used with a warning; a later lookup retries the refresh. Invalid caches are never used. No background process, scheduled job, or lookup-result cache is created.

The tool does not save queried domains, pasted URLs, registration responses, or a lookup history. Results are written only to terminal output. Persistent application data is limited to the complete IANA datasets and language preferences; cache contents do not record which WHOIS suffixes were queried.

## Development and releases

```sh
make check       # Static checks, race-enabled offline tests, formatting
make build       # Native bin/domain
make dist        # Apple Silicon and Intel archives with checksums
```

Run `go test ./internal/registry -run '^$' -bench BenchmarkParse -benchmem` to measure offline parser time and allocations. These benchmarks exclude network latency.

Offline tests verify registry-only routing, redirect/referral blocking, RDAP error handling, bulk WHOIS routing, monthly cache behavior, field extraction, language selectors, and Unicode output alignment.

`VERSION` is the single source of truth for the embedded version. Versions follow `YY.M.D.NUM`, where `NUM` starts at `1` and increments for each release on the same date. Months, days, and sequence numbers are not zero-padded. Release commits use exactly `Release YY.M.D.NUM`, without a commit body. Changes are recorded in the English [changelog](CHANGELOG.md).

Pushing a `vYY.M.D.NUM` tag runs the release workflow, checks its agreement with `VERSION` and the commit message, and publishes macOS archives and checksums with the title `Release YY.M.D.NUM` and an empty release description.

## References

- [IANA RDAP bootstrap data](https://www.iana.org/assignments/rdap-dns/)
- [IANA TLD Registry Data](https://data.iana.org/TLD/v1/tlds.json) and its [schema](https://data.iana.org/TLD/v1/tlds-schema.json)
- [RFC 7480: RDAP over HTTP](https://www.rfc-editor.org/rfc/rfc7480)
- [RFC 9083: RDAP JSON responses](https://www.rfc-editor.org/rfc/rfc9083)
- [RDAP Web Client](https://github.com/rdap-org/client.rdap.org), a reference for direct registry discovery, HTTPS endpoint preference, and the distinction between registry responses and registrar referrals. This CLI is an independent Go implementation.

## License

[MIT License](LICENSE)
