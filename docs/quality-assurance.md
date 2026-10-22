# Quality assurance

This document defines the checks required to change, build, and release Domain Check from Registries. It covers correctness, registry-only routing, privacy, security, and installation behavior.

## Supported build and verification environments

- Go 1.27.3 or newer, with dependency versions and checksums from `go.mod` and `go.sum`.
- macOS 13 (Ventura) or later for the CLI and native installation; releases include Apple Silicon and Intel executables. The installer must reject older systems before replacing an existing executable.
- macOS and Linux GitHub runners for portable Go regression tests. Linux installer tests simulate macOS system responses and use isolated paths; macOS tests exercise the actual system archive utilities.
- Python 3 for the offline installer security suite.
- Network access for the official vulnerability database and optional live checks. Core regression tests use synthetic responses and injected transports.

The CI toolchain is read from `go.mod`. Changing that version requires a fresh vulnerability scan; a clean scan using a newer local compiler does not validate an older release compiler.

## Required checks

From the checkout root:

```sh
make check
make security
make build
make dist
```

`make check` runs `go vet`, race-enabled Go tests, and formatting checks. `make security` verifies module checksums, runs the pinned official Go vulnerability scanner, and exercises the installer with synthetic downloads. These installer tests neither download releases nor change the user's installation or shell profile.

`make build` creates `build/bin/domain`; `make dist` builds both macOS architectures and SHA-256 manifests in `build/dist/`. Generated outputs remain outside the source directories and are excluded from Git.

## Regression matrix

| Area | Required behavior |
| --- | --- |
| Domain / URL input | Validate label boundaries and ASCII identity; normalize IDNs and allowed emoji; discard URL secrets locally; never fetch a pasted URL. |
| Command selection | Accept zero-to-two-hyphen selectors; save language only when standalone; keep temporary selectors temporary; show an offline introduction successfully. |
| Discovery | Use longest-suffix IANA RDAP routing, prefer HTTPS, and use bulk TLD WHOIS discovery only when RDAP is absent. |
| Registry boundary | Never follow registrar referrals, linked entities, or unresolved handles; constrain redirects to the same query at listed registry endpoints. |
| Network destinations | Reject invalid hosts, forbidden URL components, private or mixed DNS answers, and disguised local addresses; connect to the validated IPs. |
| Cache / preferences | Store only full IANA documents and language; enforce calendar-month refresh, validated stale fallback, bounded reads, private writes, and explicit partial-save errors. |
| RDAP parsing | Validate domain and Unicode identities, events, embedded roles and jCards, signing data, and response-size/depth/field limits. |
| WHOIS parsing | Verify positive and negative identities, exact labels, multiline sections, embedded objects, dates, legacy encodings, IDN families, and explicit service failures. |
| Contact attribution | Prefer a disclosed registrant organization; do not substitute administrative, technical, billing, or registrar contact values. |
| Dates | Preserve date precision, normalize recognized formats, convert only explicit zones to UTC, and exclude contact/database timestamps from domain events. |
| Output | Omit undisclosed fields; align English and Chinese by display cells; preserve grapheme clusters and safe local IDN display; neutralize terminal controls in fields and errors. |
| Availability | Keep no record, not registered, reserved, restricted, unavailable, and operational failure distinct. |
| Installation | Validate release tags, checksum manifests, expected archive members, links, duplicate paths, source arguments, replacement failures, and repeatable standard PATH setup. |

Public tests use synthetic records so real registrant information is not committed. Test transports assert the actual requested destinations and reject unauthorized requests, rather than relying only on parser output.

## Fuzz testing

Run the affected target after changes to normalization, parsing, or terminal sanitization:

```sh
go test ./internal/registry -run '^$' -fuzz FuzzDomainCannotInjectWHOISOrURL -fuzztime 10s
go test ./internal/registry -run '^$' -fuzz FuzzUntrustedRegistryResponses -fuzztime 10s
go test ./internal/registry -run '^$' -fuzz FuzzCleanPreservesSanitization -fuzztime 10s
```

The domain target checks that accepted normalized identities cannot become additional WHOIS commands or URL components. The response target exercises both parsers with hostile input. The sanitization target checks control-character neutralization. A bounded fuzz run is evidence for the executed cases, not an exhaustive proof.

## Registry-format verification

Service inventories must include every active IANA suffix with WHOIS and no RDAP entry, including `.edu`, `.arpa`, and IDN suffixes across scripts. A two-letter country-code filter is insufficient. Compare independently identified benign registered names, registry-operated names with exceptional formats, and explicit negative candidates. A candidate's spelling does not establish that it is unregistered.

Private developer samples may be retained separately for regression analysis. They are ignored by Git, never loaded by the CLI, and must not be copied into public fixtures or reports. Parser changes must be compared against the reviewed corpus baseline; every intended output difference requires inspection before the baseline is updated. Connection failures and access restrictions must remain distinct from parsed negative answers.

The published [coverage snapshot](domains.md) states its source date and verification limits. Format support does not imply current reachability or complete ownership disclosure.

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
2. Complete the required checks with the release toolchain. Investigate vulnerability findings; do not disable scanning to publish.
3. Verify both architecture builds, archive layout, and checksum manifests. Keep the public documentation and installation paths consistent.
4. Commit with exactly `Release YY.M.D.NUM`, push the commit, and push its matching `vYY.M.D.NUM` tag.
5. Require successful check and release workflows and verify the actual downloadable assets before announcing publication. A failed build or a pushed tag alone is not a published release.

Workflow actions are pinned to full commit hashes. Check jobs have read-only repository permissions; only the release job receives the permission needed to publish assets. Release descriptions are empty.

## Limits of the assurance

Registry responses, access rules, and IANA service advertisements can change. A passing offline suite does not prove current service availability, registration eligibility, or the accuracy of ownership disclosures. WHOIS and HTTP-only services retain their transport limitations. Vulnerability scanning identifies known reachable findings available at scan time; additional review and future fixes may still be necessary. See [security.md](security.md) for the trust model.
