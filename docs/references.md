# References

These sources explain the discovery data, RDAP behavior, and reference client that informed Domain Check from Registries. Each source is described in relation to the implementation it informs.

## IANA RDAP bootstrap data

The [IANA RDAP DNS bootstrap registry](https://www.iana.org/assignments/rdap-dns/) publishes the suffix-to-service assignments distributed in [dns.json](https://data.iana.org/rdap/dns.json). Domain Check uses that complete dataset to decide whether the queried suffix has RDAP and which registry endpoints are authoritative. HTTPS endpoints are preferred when advertised.

The bootstrap is the routing source of truth. Domain Check retains RDAP routing when an advertised service returns an error and caches the complete document locally.

## IANA TLD Registry Data

[IANA TLD Registry Data](https://data.iana.org/TLD/v1/tlds.json) and its [schema](https://data.iana.org/TLD/v1/tlds-schema.json) describe top-level domains, their status, and published WHOIS hosts. Domain Check uses the WHOIS hostname for an active suffix only when the RDAP bootstrap has no matching service. RDAP fields in this dataset do not override dns.json.

Downloading the complete document avoids asking IANA about a specific queried suffix. The public coverage inventory in [domains.md](domains.md) is a dated snapshot of these discovery sources; actual routing uses the locally validated datasets and their update policy.

## RDAP over HTTP

[RFC 7480: HTTP Usage in the Registration Data Access Protocol](https://www.rfc-editor.org/rfc/rfc7480) provides the HTTP foundation for RDAP requests and responses. Domain Check uses HTTP status handling to distinguish a missing record from authentication, access, rate-limit, and service failures.

The project's registry-only policy additionally constrains redirects to the same domain query at an IANA-listed registry endpoint. This policy keeps each request at the authoritative registry throughout the lookup.

## RDAP JSON responses

[RFC 9083: JSON Responses for the Registration Data Access Protocol](https://www.rfc-editor.org/rfc/rfc9083) describes the domain objects, events, entities, and nameservers used to interpret a registry response. These structures supply the fields normalized into the CLI's common output model.

Domain Check verifies the returned domain identity before displaying registration details. It reads disclosed contact information from embedded entities and does not retrieve linked objects to fill missing rows. Registry omissions and redactions remain omissions in the output.

## RDAP Web Client

[RDAP Web Client](https://github.com/rdap-org/client.rdap.org) provided a reference for direct registry discovery, HTTPS endpoint preference, and the distinction between registry responses and registrar referrals. It helped frame the registry-first behavior while Domain Check implements its own Go CLI, parsing, local state, and terminal presentation.

Domain Check is an independent Go implementation distributed under the [MIT License](../LICENSE). Its architecture and lookup flow are documented in [technical.md](technical.md).
