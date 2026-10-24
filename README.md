# Domain Check from Registries

A macOS command-line tool that looks up domain registration details directly from registries and returns them in a clean, readable layout.

```sh
domain example.com
domain https://www.example.com/path
```

Type a domain name or paste a full URL copied from your browser. The command picks out the domain and prints its key registration details in one aligned view. An optional [local web interface](#local-web-interface) brings the same lookups to your browser. Domain Check supports [1,367 top-level domains](docs/domains.md).

Visit the [product website](https://sd800.github.io/domain-check/).

## Product statement

Domain Check from Registries is built around a simple principle: **a domain lookup should go straight to the source and come back easy to read.** It works right on your Mac, in Terminal or your browser, from query out to answer back.

**Send the query straight to the registry.** Searching through commercial domain websites or following registrar referrals exposes interest in a domain name to tracking, giving others a signal to front-run it or hold onto it. Domain Check queries the registry's own RDAP and WHOIS services and never follows registrar referrals. Whether the input is a domain name or a full URL, every lookup goes directly to the registry and nowhere else.

**Bring back a clean, readable answer.** Registry responses often contain inconsistent formatting, redundant fields, and technical details that obscure useful information. Domain Check extracts relevant registration details, organizes them into a consistent, easy-to-scan format, and drops the clutter. Whether the response is RDAP or registry-specific WHOIS, every answer comes back in the same clean layout.

**One command, one clean answer, straight from the registry.**

## Install

Requires macOS 13 (Ventura) or later.

Copy this entire block into macOS Terminal. It installs the latest release for Apple Silicon or Intel, verifies its SHA-256 checksum, and makes `domain` available immediately. No Go installation or manual download is required.

```sh
/bin/sh <<'SH' && export PATH="$HOME/.local/bin:$PATH" && domain --version
set -eu
PATH=/usr/bin:/bin:/usr/sbin:/sbin
export PATH
umask 077
installer=$(mktemp)
trap 'rm -f "$installer"' EXIT
curl -q --proto '=https' --proto-redir '=https' --connect-timeout 15 \
  --max-time 180 --max-filesize 1048576 -fsSL --retry 3 -o "$installer" \
  https://raw.githubusercontent.com/sd800/domain-check/main/scripts/install.sh
sh "$installer"
SH
```

The [commented installer](scripts/install.sh) puts the executable in `~/.local/bin` and adds a standard `PATH` entry to `${ZDOTDIR:-$HOME}/.zshrc` only once. The block above also sets `PATH` in the current terminal and prints the installed version; no restart is needed. Run the same block again to install the latest release.

Archives and checksum files are also available on the [Releases page](https://github.com/sd800/domain-check/releases).

<details>
<summary>Build and install from source</summary>

Requires Go 1.27.3 or newer and Git. Run this block from the directory where you want the checkout. It builds the executable, uses the same installer for installation and future zsh sessions, then enables `domain` in the current terminal:

```sh
git clone https://github.com/sd800/domain-check.git &&
cd domain-check &&
make build &&
sh scripts/install.sh build/bin/domain &&
export PATH="$HOME/.local/bin:$PATH" &&
domain --version
```

To build without installing, run `make build` and use `./build/bin/domain`.

</details>

## Usage

```sh
domain                           # Version and a short introduction; exit code 0
domain example.com               # Saved language, or device language initially
domain -en example.com           # English for this query
domain zh example.com            # Simplified Chinese for this query
domain --zh example.com          # Simplified Chinese for this query
domain auto-lang example.com     # Device language for this query
domain example.com en            # Selectors also work after the domain
domain cn                        # TLD allocation details from IANA RDAP
domain .com                      # A leading dot is accepted
domain .github.io                # Queries github.io
domain auto                      # Queries the .auto TLD
domain zh                        # Save Simplified Chinese as the default
domain en                        # Save English as the default
domain auto-lang                 # Restore device-language detection
domain update                    # Refresh IANA RDAP and WHOIS data immediately
domain --update                  # Same explicit refresh
domain setup                     # Configure the optional local web interface
domain uninstall                 # Uninstall after an explicit confirmation
domain 例子.中国                  # Internationalized domain name
domain 'https://www.example.com/path?item=1&lang=zh'
domain --help
domain --version
```

Running `domain` without arguments prints the program name, version, copyright notice, a short introduction, and usage examples in the default language, then exits successfully. This is an offline installation smoke test and does not update the IANA cache. To also verify network access, run `domain example.com`.

Use `domain en` or `domain zh` to switch and save the default language, or `domain auto-lang` to restore device-language detection. These standalone language commands print a short confirmation after the preference has been saved, without performing a lookup or updating the IANA cache.

Language selectors accept zero, one, or two leading hyphens:

| Language | Selectors |
| --- | --- |
| English | `en`, `-en`, `--en` |
| Simplified Chinese | `zh`, `-zh`, `--zh` |
| Device language | `auto-lang`, `-auto-lang`, `--auto-lang` |

The `update`, `setup`, `uninstall`, `help`, and `version` commands also accept zero, one, or two leading hyphens. For example, `domain update`, `domain -update`, and `domain --update` perform the same refresh. `-h` and `-v` remain available as shortcuts for help and version. Language selectors work with these commands, such as `domain zh update`. Run `update`, `setup`, and `uninstall` without a domain or URL.

The last language selector wins for the current invocation. When supplied with a domain, URL, `help`, `version`, `update`, `setup`, or `uninstall`, it applies only to that invocation and does not change the saved default. Only standalone `domain en`, `domain zh`, or `domain auto-lang` commands save the default language. With no saved selection, or after standalone `domain auto-lang`, the command uses the Mac's first preferred language, with Chinese mapped to Simplified Chinese and other languages mapped to English. In automatic mode, explicit `LC_ALL` or `LC_MESSAGES` values override the device preference; `LANG` is the fallback when the preference cannot be read.

Preferences are stored at `~/Library/Application Support/domain-check/config.json` on macOS. Set `DOMAIN_CONFIG_DIR` to use another configuration directory. Preferences are separate from the IANA cache, so deleting cached bootstrap data does not reset the language. Invalid arguments do not change the saved preference. If a standalone language command cannot save its preference, it reports the failure and returns exit code `3`. If reading a saved preference fails, other commands can continue in the device language with a warning. An explicit language selector bypasses reading the saved preference.

Pass one domain or URL per invocation. URLs are automatically reduced to their registrable domain: `https://www.example.com:8443/path?item=1` queries `example.com`, and `https://www.example.co.uk/` queries `example.co.uk`. Addresses without a scheme, such as `www.example.com/path`, are also accepted. The URL's scheme, credentials, port, path, query, and fragment are not sent to the registry, and the URL itself is never fetched. Private hosting suffixes are reduced to the registry-level domain; for example, `https://user.github.io/` queries `github.io`.

Quote URLs containing shell-special characters such as `?`, `&`, `*`, or parentheses, as shown above, so the shell passes the complete URL to `domain`.

Direct domain inputs retain their normalized identity. One leading dot is ignored, so `domain .github.io` queries `github.io`; a trailing dot is also accepted. Single-label names query the TLD itself: `domain cn` and `domain .cn` both retrieve its allocation record directly from IANA RDAP. Bare command names take priority, so `domain setup` opens settings while `domain .setup` queries that TLD name. `auto` now queries the `.auto` TLD; use `auto-lang` for device-language detection. Internationalized names, including emoji and joined emoji sequences, are converted to ASCII/Punycode for querying. The domain line shows that ASCII name, with a separate Unicode display name when it can be decoded and validated.

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

注册时间:     2003 年 3 月 10 日 11:19:18
过期时间:     2027 年 5 月 15 日 11:25:25
更新时间:     2025 年 5 月 15 日 10:25:25

注册商:       Example Registrar
状态:         ok

注册人:       Example Organization
联系邮箱:     admin@example.com

域名服务器:   ns1.example.com
域名服务器:   ns2.example.com
DNSSEC:       unsigned
```

Only fields supplied by the registry are shown. Missing, empty, or explicitly redacted values in the registry's response are omitted. A registry may disclose no registrant name or email; the command does not contact a registrar to fill those gaps. Registry status values and names retain their source language. For ordinary domain RDAP results, a two-letter country code disclosed in the selected registrar's address is appended with one space and square brackets, such as `Example Registrar [US]`. No country is inferred, and TLD allocation results receive no country annotation. For `.edu` WHOIS records, the Registrant row also includes the disclosed department and postal lines, with each continuation aligned beneath the institution name.

For `.gov` RDAP records, disclosed registrant address details appear directly below Registrant: Address, City, State, Postal Code, and Country. Missing or redacted components are omitted. These details come from the selected registrant contact, and the country retains the registry's wording.

Other registrant contacts with disclosed address details show separately supplied names and organizations as Registrant and Organization. When only an organization is disclosed, it appears as Registrant. Street, City, State/Province, Postal Code, Country, Phone, and Email follow when supplied. A whole Address or Addr value stays in a multiline Address row; it is not split into inferred components. Missing and redacted values are omitted, and administrative, technical, and registrar contacts are not substituted. Established registry-specific address layouts, including `.edu`, `.gov`, and TLD allocation records, remain unchanged.

WHOIS parsing supports registry-specific labels, contact sections, linked contact objects included in the same response, and multiline nameserver and DNSSEC records. Format checks include non-country namespaces such as `.edu` and `.arpa`, and internationalized top-level domains across scripts. Explicit policy notices remain distinct from registered records and service failures. Domain dates remain separate from contact-object dates, and administrative, technical, and registrar email addresses are not substituted for a registrant email. Access restrictions, timeouts, and unrecognized responses are reported as service errors, keeping them distinct from registration results.

Content begins in a fixed column determined by the longest label, with three spaces after that label's colon. Chinese widths use terminal display cells; `DNSSEC` does not determine the Chinese column. Long values wrap to the terminal width, with continuation lines starting in the content column. When terminal dimensions are unavailable, `COLUMNS` is used, falling back to 80 columns. Very narrow terminals must still accommodate the label column and at least one character.

English dates use `YYYY-MM-DD`, including values returned as `23-May-1985`, and timestamps use `YYYY-MM-DD HH:MM:SS`. Chinese output uses `YYYY 年 M 月 D 日`, followed by the original `HH:MM:SS` when supplied. Explicitly zoned timestamps are converted to UTC and labeled with a ` UTC` suffix. Unzoned timestamps retain their source wall time without a timezone label; date-only records retain date precision. Approximate or unrecognized date text is preserved. The RDAP database refresh timestamp is not substituted for a domain's last-changed time. DNSSEC is shown only when the registry supplies enough information to determine it.

### TLD allocation records

TLD queries use IANA's root-registry RDAP service directly, in both the CLI and browser. Their layout places status directly after the last-changed date, followed by the registrant and postal address, registration URL, administrative contact, and delegation details when disclosed. The administrative contact is selected by the RDAP `administrative` role; its separate `Role` row displays the contact's jCard role. Voice and fax numbers appear as individual `Phone` rows. Multiline addresses retain their line structure and align with the value column.

All result values are plain text, including URLs, email addresses, and telephone numbers. Domain Check does not create hyperlinks or fetch registration URLs.

### Missing records and registration restrictions

```text
Domain Name:         example.com
Result:              No registry record
```

```text
域名:         example.com
查询结果:     注册局无记录
```

`Registry did not provide domain-specific registration data` (`注册局未提供此域名的独立注册信息`) identifies a namespace-only reply or a recognized non-disclosure response, including the observed `.edu.tw`, `.gov.tw`, and `.mil.tw` cases. It makes no availability claim and does not substitute the parent namespace's manager for the queried domain's registrant.

By default, an RDAP `404` is displayed as `No registry record`; for a TLD query, it is `No IANA TLD record` (`IANA 无此顶级域名记录` in Chinese). An absent TLD record makes no claim that the name can be registered. Disclosed TLD statuses such as `inactive` or `reserved` remain visible alongside the allocation details. Registration availability and restrictions are reported through distinct results: `Not registered`, `Reserved`, and `Registration not allowed` reflect explicit registry statements. `Not available for registration` preserves a registry availability restriction, while `Registration restricted (online registration unavailable)` identifies an online-registration restriction. `Domain usage restricted` preserves an explicit registry usage restriction. `Registration eligibility restricted` identifies a disclosed qualification requirement. Timeouts, access denials, rate limits, unsupported suffixes, and malformed responses are reported as errors on stderr.

| Exit code | Meaning |
| --- | --- |
| `0` | A registry record was returned, language preferences or IANA data were saved, or setup, the introduction, help, or version finished successfully |
| `1` | No registry record, the registry did not provide domain-specific data, or an explicit registration restriction |
| `2` | Invalid arguments or domain |
| `3` | Discovery, configuration, network, service, response, or output error |

## Local web interface

Enable the optional local web interface through the setup menu:

```sh
domain setup
```

On the first screen, choose **1 Enable** to start the interface and open [domain-check.localhost:20771](http://domain-check.localhost:20771), then choose **3 Next**. On the second screen, choose **1 Enable** to start the interface automatically when you log in to macOS, then **3 Finish**. To turn automatic startup off, choose **2 Disable**, then **3 Finish**. Both options show their current status. Automatic startup is off initially; disabling it keeps the interface available for the current login session. Disabling the web interface also disables automatic startup.

The menu also accepts `domain -setup` and `domain --setup`. It uses your account's LaunchAgent and requires no administrator privileges, additional runtime, or hosts-file changes. Run it again to change either setting. Install the executable before enabling the interface so its path remains available.

Type a domain or paste a URL in the search box, or open a direct lookup:

```text
http://domain-check.localhost:20771/example.com
http://domain-check.localhost:20771/.com
http://domain-check.localhost:20771/.github.io
http://domain-check.localhost:20771/https%3A//www.google.com
```

The interface follows your device's light or dark appearance and browser language initially. The controls at the top switch between system, light, and dark themes, and between English and Simplified Chinese. These browser preferences are saved independently of the CLI language. All assets are served locally, and text uses system fonts without remote font requests. Long results scroll naturally with the page, and existing results stay visible while the next query runs.

The service listens only on your Mac's loopback addresses. It does not expose the interface to your local network or store lookup history. Updates through the installer or `make install` refresh a loaded interface through its existing LaunchAgent, preserving its settings and startup registration. Repeating an enabled setup option keeps that registration intact. Run `domain uninstall` to remove the tool; it stops the interface and removes the login registration after confirmation.

## Uninstall

Run `domain uninstall`, `domain -uninstall`, or `domain --uninstall`. The command asks for confirmation and defaults to cancel. Enter `y` or `yes` and press Return to proceed; a blank answer, a negative answer, end-of-input, or an interrupt cancels without changing the installation. Language selectors remain temporary, for example `domain zh uninstall`.

After confirmation, Domain Check stops its current-user web service and removes its login registration, this installation's executable and command link, CLI and web settings, and IANA cache files. It requires no administrator access. Source files, developer samples, unrelated files, containing directories, and the standard shell PATH entry are preserved. Browser appearance and language preferences remain in the browser. If a service or filesystem operation fails, the command reports the failure; it does not request elevated access.

## Registry-only queries

- TLD allocation queries go directly to `https://rdap.iana.org/domain/<tld>`, without fetching discovery datasets.
- `.tw` is classified as WHOIS-only; domain queries use the registry WHOIS host from the complete IANA TLD dataset directly. The `.tw` TLD allocation query itself still uses IANA RDAP.
- Delegated `.ac.uk` and `.gov.uk` names use their documented namespace registries directly: Jisc WHOIS and Nominet's dedicated RDAP service, respectively.
- Other ordinary domain RDAP requests use endpoints listed by IANA for the domain's matching suffix, preferring HTTPS.
- Response links and entity links are never fetched, including registrar `related` links.
- HTTP redirects are accepted only for the same domain query within the selected registry endpoints, including an HTTP-to-HTTPS upgrade. Arbitrary paths, external hosts, and redirects with query parameters are blocked.
- When RDAP is absent, WHOIS uses the registry host in IANA's complete `tlds.json` dataset, then queries that registry directly on TCP port 43. Registrar WHOIS fields and referrals in the registry response are ignored. The `rdap_base_urls` field in `tlds.json` does not override `dns.json`.
- RDAP-enabled domains use registry WHOIS as a fallback after an RDAP connection, service, or response failure, when the complete IANA TLD dataset identifies an active suffix with a WHOIS host, or the fixed Nominet-published `.uk` service applies. This covers country-code and generic TLDs. Successful replies, including missing fields and no-record results, are kept as returned. Rate limits, blocked redirects, and cancellation do not trigger fallback.

WHOIS parsing combines shared field extraction with registry-specific format rules, including CNNIC and Nominet-style records. Labeled fields, multiline sections, and explicit service responses are processed consistently; unrecognized replies produce a clear error. WHOIS requires outbound TCP port 43. RDAP and WHOIS results use the same layout, without combining records from the two protocols.

## IANA cache

The tool caches two complete, official IANA datasets on macOS:

```text
~/Library/Caches/domain-check/dns.json
~/Library/Caches/domain-check/tlds.json
```

`dns.json` determines RDAP routing. The complete TLD registry in `tlds.json` supplies WHOIS server addresses; entries are not selected or written according to queried suffixes. Set `DOMAIN_CACHE_DIR` to use another cache directory.

Cache freshness is checked only during a lookup, when the corresponding dataset is needed. TLD allocation queries go straight to IANA RDAP and do not read or update either cache. After one calendar month from its last successful fetch, the next lookup that needs that file refreshes it. Month-end dates are clamped: a January 31 fetch becomes due on February 28, or February 29 in a leap year. The IANA publication date does not determine local cache age. The introduction, help, version, setup, and standalone language commands do not fetch or refresh IANA data.

Run `domain update` to download and validate both current datasets immediately, even when the local caches are fresh. Both downloads are validated before either cached file is replaced. Each replacement is atomic and restarts that file's one-month freshness period. Download, validation, or cache-write failures return exit code `3`; an incomplete update never reports success. If saving one file succeeds before saving the other fails, that file remains refreshed and the other keeps its previous contents.

If an automatic refresh fails, a previously validated cache is used with a warning; a later lookup retries the refresh. Invalid caches are never used. IANA refreshes are driven by lookups or `domain update`; enabling the optional web service does not schedule metadata downloads. No lookup-result cache is created.

The tool does not save queried domains, pasted URLs, registration responses, or a lookup history. Results are displayed in the Terminal or the optional local web interface. Persistent application data contains the complete IANA datasets, language preference, and optional web-service settings. The browser stores only its theme and language preferences. Cache contents do not record which WHOIS suffixes were queried.

## Development and releases

```sh
make check       # Static checks, race-enabled offline tests, formatting
make security    # Module verification, vulnerability scan, installer attack tests
make build       # Native build/bin/domain
make dist        # Apple Silicon and Intel archives with checksums
```

Run `go test ./internal/registry -run '^$' -bench BenchmarkParse -benchmem` to measure offline parser time and allocations. These benchmarks exclude network latency.

Offline tests verify registry-only routing, redirect/referral blocking, RDAP error handling, bulk WHOIS routing, monthly cache behavior, field extraction, language selectors, and Unicode output alignment.

`VERSION` is the single source of truth for the embedded version. Versions follow `YY.M.D.NUM`, where `NUM` starts at `1` and increments for each release on the same date. Months, days, and sequence numbers are not zero-padded. Release commits use exactly `Release YY.M.D.NUM`, without a commit body. Changes are recorded in the English [changelog](CHANGELOG.md).

Pushing a `vYY.M.D.NUM` tag runs the release workflow, checks its agreement with `VERSION` and the commit message, and publishes macOS archives and checksums with the title `Release YY.M.D.NUM` and an empty release description.

## Documentation

- [Technical details and implementation](docs/technical.md)
- [Domain suffixes and registry coverage](docs/domains.md)
- [Security policy and safeguards](docs/security.md)
- [Quality assurance and verification](docs/quality-assurance.md)
- [References](docs/references.md)

## License

[MIT License](LICENSE)

## Privacy policy

Domain Check from Registries is designed to keep your domain lookups private.

**Lookups are processed on your Mac.** Domain Check has no remote servers, accounts, or analytics. It prepares each query, formats the registry's response, and displays the result in your Terminal or optional local browser interface. The browser interface runs only on your Mac. When you paste a URL, Domain Check reduces it to its registrable domain on your Mac before sending a query.

**Minimal data is stored.** Domain Check stores only your CLI language preference, two complete IANA service-discovery datasets, and settings for the optional local web service. The browser stores only your theme and language choices. Queried domains, pasted URLs, and registry responses are held in memory for the current lookup only and are not saved by Domain Check. The local web service keeps no access logs. Your shell and browser may keep their own command or navigation history, according to their settings.

**Queries go directly to registries.** Except as described below for `.edu.cn`, the queried domain is sent only to the registry responsible for it. Domain Check does not contact registrars or follow registrar referrals. Registries receive the queried domain and standard connection information, such as your IP address. For ordinary domain lookups, IANA's service-discovery datasets are downloaded in full without disclosing the queried domain or suffix to IANA. When you query a TLD itself, IANA acts as its root registry and receives that TLD query directly. Queries use the registry's RDAP or WHOIS service, preferring HTTPS for RDAP when advertised by the registry.

**Third-party services.** Registries, IANA, GitHub, and DNS providers are independent services, and their own privacy policies govern how they handle your information. Domain Check does not control their practices. Installing or updating Domain Check downloads files from GitHub. As with any internet connection, network operators along the path may be able to observe some traffic, including WHOIS queries, which use a standard protocol without encryption.

**`.edu.cn` lookups.** CERNET (China Education and Research Network), the registry for `.edu.cn`, generally makes its WHOIS service available only within the CERNET network. To provide useful results for these domains, Domain Check first queries the registry and, if its service is unavailable, retrieves the domain's public nameserver records through your configured DNS resolver, which then receives the domain name. Registration under `.edu.cn` is reserved for eligible educational and research institutions in mainland China and requires CERNET approval, which substantially reduces the risk of front-running and speculative holding.
