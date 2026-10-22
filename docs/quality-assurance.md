# Quality assurance

This document defines the checks required to change, build, and release Domain Check from Registries. It covers correctness, registry-only routing, privacy, security, and installation behavior.

## Supported build and verification environments

- Go 1.27.3 or newer, with dependency versions and checksums from `go.mod` and `go.sum`.
- macOS 13 (Ventura) or later for the CLI and native installation; releases include Apple Silicon and Intel executables. The installer must reject older systems before replacing an existing executable.
- macOS and Linux GitHub runners for portable Go regression tests. Linux installer tests simulate macOS system responses and use isolated paths; macOS tests exercise the actual system archive utilities.
- Python 3 for the offline installer security suite.
- Network access for the official vulnerability database and optional live checks. Core regression tests use synthetic responses and injected transports.

CI reads the release toolchain from `go.mod` and runs vulnerability scanning with that toolchain. Toolchain changes receive the same validation before release.

## Required checks

From the checkout root:

```sh
make check
make security
make build
make dist
```

`make check` runs `go vet`, race-enabled Go tests, and formatting checks. `make security` verifies module checksums, runs the pinned official Go vulnerability scanner, and exercises the installer in isolated directories with synthetic release data.

`make build` creates `build/bin/domain`; `make dist` builds both macOS architectures and SHA-256 manifests in `build/dist/`. Generated outputs remain outside the source directories and are excluded from Git.

## Regression matrix

| Area | Required behavior |
| --- | --- |
| Domain / URL input | Validate label boundaries and ASCII identity; normalize IDNs and allowed emoji; discard URL secrets locally; never fetch a pasted URL. |
| Command selection | Accept zero-to-two-hyphen selectors; save language only when standalone; keep temporary selectors temporary; show an offline introduction successfully. |
| Discovery | Use longest-suffix IANA RDAP routing, prefer HTTPS, and use bulk TLD WHOIS discovery only when RDAP is absent. |
| Registry boundary | Never follow registrar referrals, linked entities, or unresolved handles; constrain redirects to the same query at listed registry endpoints. |
| Network destinations | Reject invalid hosts, forbidden URL components, private or mixed DNS answers, and disguised local addresses; connect to the validated IPs. |
| Cache / preferences | Store only full IANA documents, language, and optional web settings; enforce calendar-month refresh, validated stale fallback, bounded reads, private writes, and explicit partial-save errors. |
| RDAP parsing | Validate domain and Unicode identities, events, embedded roles and jCards, signing data, and response-size/depth/field limits. |
| WHOIS parsing | Verify positive and negative identities, exact labels, multiline sections, embedded objects, dates, legacy encodings, IDN families, and explicit service failures. |
| Contact attribution | Prefer a disclosed registrant organization; do not substitute administrative, technical, billing, or registrar contact values. |
| Dates | Preserve date precision, normalize recognized formats, use Chinese year/month/day presentation only for Chinese output, convert only explicit zones to UTC, and exclude contact/database timestamps from domain events. |
| Output | Omit undisclosed fields; align English and Chinese by display cells; preserve grapheme clusters and safe local IDN display; neutralize terminal controls in fields and errors. |
| Availability | Keep no record, not registered, reserved, restricted, unavailable, and operational failure distinct. |
| Local web API | Accept only loopback clients, allowlisted Hosts, same-origin metadata, JSON and the session token; enforce body, rate, concurrency and time limits; preserve the registry-only engine. |
| Web rendering | Render remote fields as text, omit missing values, preserve dates and IDN identities, retain the old record while pending, and keep queries out of storage and logs. |
| Setup / service | Verify all setup selector forms and both menus, saved status, current-session operation, login startup, disabling, port conflicts, update refresh, and failed-transition preservation. |
| Installation | Validate release tags, checksum manifests, expected archive members, links, duplicate paths, source arguments, replacement failures, and repeatable standard PATH setup. |

Public tests use synthetic records to protect registrant privacy. Test transports verify the actual requested destinations and reject unauthorized requests alongside the parser assertions.

## Browser and macOS service verification

When changing the browser interface, run the server from a native build and inspect it in a current browser at `http://domain-check.localhost:20771`. Use synthetic records for screenshots and deterministic browser checks. Verify desktop and phone widths, both languages, system/light/dark appearance, and saved preferences on reload. Confirm a 15px body base, 14px field labels, 11px copyright, normal letter spacing, no remote fonts or assets, keyboard submission, stable search-box borders on focus, visible button focus, and natural page scrolling for long records without horizontal overflow.

Exercise direct paths containing a domain, a partially encoded full URL, and a fully encoded URL. Confirm URL normalization through the shared registry engine, safe rendering of HTML-shaped text, explicit negative results, and useful errors. A new query must retain the current result while pending; theme switches must preserve result nodes. The console must show no script or CSP errors. Browser storage must contain only appearance and language choices.

On macOS, test the actual two-step setup menu with web enable/disable and login startup enable/disable. Inspect the user LaunchAgent registration, verify both loopback listeners, and confirm the service survives closing the terminal. Check that disabling removes registrations, that no administrator prompt or hosts-file change occurs, and that executable installation refreshes a previously enabled service. Restore the desired user settings after the checks.

## Fuzz testing

Run the affected target after changes to normalization, parsing, or terminal sanitization:

```sh
go test ./internal/registry -run '^$' -fuzz FuzzDomainCannotInjectWHOISOrURL -fuzztime 10s
go test ./internal/registry -run '^$' -fuzz FuzzUntrustedRegistryResponses -fuzztime 10s
go test ./internal/registry -run '^$' -fuzz FuzzCleanPreservesSanitization -fuzztime 10s
```

The domain target checks that accepted normalized identities cannot become additional WHOIS commands or URL components. The response target exercises both parsers with hostile input. The sanitization target checks control-character neutralization. These targets complement the deterministic regression suite with generated inputs.

## Registry-format verification

Service inventories must include every active IANA suffix with WHOIS and no RDAP entry, including `.edu`, `.arpa`, and IDN suffixes across scripts. Coverage includes both country and non-country namespaces. Compare independently identified benign registered names, registry-operated names with exceptional formats, and explicit negative candidates. Negative results are classified from the registry's explicit response.

Private developer samples may be retained separately for regression analysis. They are ignored by Git, never loaded by the CLI, and must not be copied into public fixtures or reports. Parser changes must be compared against the reviewed corpus baseline; every intended output difference requires inspection before the baseline is updated. Connection failures and access restrictions must remain distinct from parsed negative answers.

The published [coverage snapshot](domains.md) records the source date, observed responses, and positive-record coverage. Service access and disclosed fields are assessed separately from format handling.

## Optional live smoke checks

```sh
./build/bin/domain
./build/bin/domain --version
./build/bin/domain en example.com
```

The first two checks are offline. The final command verifies a current registry connection and standardized output. Additional live checks should be bounded, use benign names, and respect registry rate limits. Network-dependent checks are separate from deterministic regression tests.

Verify archive contents and checksums before distribution:

```sh
cd build/dist
shasum -a 256 -c "checksums_$(cat ../../VERSION).txt"
```

Archives must contain one regular `domain` executable, the root release documents, and only the expected public documentation paths. Failed verification must not replace an installed executable.

## Release gate

1. Update root `VERSION` using `YY.M.D.NUM` and add an English changelog entry.
2. Complete the required checks with the release toolchain. Resolve actionable vulnerability findings before publication.
3. Verify both architecture builds, archive layout, and checksum manifests. Keep the public documentation and installation paths consistent.
4. Commit with exactly `Release YY.M.D.NUM`, push the commit, and push its matching `vYY.M.D.NUM` tag.
5. Confirm successful check and release workflows, then verify the downloadable assets before announcing publication.

Workflow actions are pinned to full commit hashes. Check jobs have read-only repository permissions; only the release job receives the permission needed to publish assets. Release descriptions are empty.

## Ongoing validation

Deterministic tests validate routing, parsing, privacy, and installation behavior. Live checks assess current registry access, while dated coverage snapshots document observed formats and responses. Every release also checks reachable code against the official Go vulnerability database. Registry changes and new findings inform subsequent regression coverage and updates. See [security.md](security.md) for the implemented controls and protocol details.
