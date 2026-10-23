# Domain suffixes and registry coverage

Domain Check accepts domain names and full URLs, including internationalized labels. Routing follows the current IANA datasets. This inventory records their service assignments and the corresponding registry-format verification.

## TLD allocation queries

Query a TLD itself with `domain cn` or `domain .com`; internationalized TLDs are accepted in Unicode or Punycode. These allocation queries go directly to IANA RDAP, independently of the domain-registration service coverage below. IANA's disclosed allocation record supplies the result, including any inactive or reserved status. No record means no IANA TLD record, without implying that the name is available for registration.

One leading dot is also accepted on ordinary domains: `domain .github.io` queries `github.io`. Bare command names take priority; use a leading dot to query a matching TLD name.

## IANA snapshot

Source: the complete [IANA TLD Registry Data](https://data.iana.org/TLD/v1/tlds.json), generated **2026-10-21T21:54:03+00:00**, and [IANA RDAP DNS bootstrap data](https://data.iana.org/rdap/dns.json), published **2026-09-30T23:00:03Z**. The TLD registry's RDAP fields do not replace the DNS bootstrap as the RDAP routing source.

| Active top-level suffixes, by lookup route | Count |
| --- | ---: |
| RDAP primary | 1202 |
| WHOIS-only, including `.tw` | 165 |
| Neither protocol published | 70 |
| Total active TLDs in this snapshot | 1437 |

These services cover **1,367 TLDs**, including **141 internationalized TLDs**. IANA advertises RDAP for 1203 TLDs; Domain Check classifies `.tw` as WHOIS-only because its RDAP interface is unavailable, while retaining both complete IANA files unchanged.

WHOIS-only coverage includes 120 ordinary ASCII TLDs and 45 internationalized TLDs. IDN classification refers to the top-level suffix itself.

Of the **77 RDAP-primary ccTLDs**, **75** have a registry WHOIS fallback: 74 from IANA and the fixed Nominet-published `.uk` service. The remaining 2 (`.na`, `.pn`) have no published WHOIS service in this inventory. Fixed delegated-namespace services, including Jisc `.ac.uk`, retain their documented routes.

Run `domain update` to refresh the complete IANA datasets. Service inventories and observed availability are recorded separately.

## RDAP suffixes

The full advertised RDAP list is available in the [official DNS bootstrap](https://data.iana.org/rdap/dns.json). Domain Check selects the longest matching suffix and prefers HTTPS. RDAP-primary ccTLDs may use their published registry WHOIS service after a connection, service, or response failure. Successful replies, including no-record responses and missing fields, complete the lookup. Rate limits, blocked redirects, and cancellation stop it. Other TLDs retain RDAP without WHOIS fallback; registrar referrals are never followed.

## Explicit `.tw` WHOIS routing

Domain Check queries the [IANA-published TWNIC WHOIS service](https://www.iana.org/domains/root/db/tw.html) for names under `.tw`. Its parser covers ordinary and multilabel registrations, comma-delimited statuses, registrant contact lines, UTC+8 timestamps, reserved namespace names, and negative responses.

[TWNIC's namespace policy](https://twnic.tw/dnservice/policy/contents.php?id=4&lang=zh-tw) assigns `.edu.tw`, `.gov.tw`, and `.mil.tw` to separate public authorities. A parent-only education record, government referral, or recognized military-namespace refusal is displayed as `Registry did not provide domain-specific registration data`; none establishes the child's owner or availability. The [education authority's public WHOIS link](https://depart.moe.edu.tw/ed2700/cp.aspx?n=2EA55D7D4B421245&s=EC480F252DFCA938) is identified as an IPv4 management lookup. The [government registry service](https://gsn.nat.gov.tw/GSNFAQ/FAQContents/6) documents a separate registration portal. These public references are distinct from a verified domain-query API.

## Education, government, and other special namespaces

The 2026-10-22 review inventories `edu`, `ac`, `gov`, and `mil` beneath all **309 active country-code TLDs**, plus additional ICANN public-suffix categories for schools, local government, justice, health, research, professional bodies, and other institutional uses. The inventory contains **1,780 candidate namespaces**: 1,236 from the four core labels and 544 additional categories. A candidate or public-suffix entry describes a name boundary; it does not establish registration eligibility or availability.

Verification compares namespace records, generated absence probes, and real institutional names selected from search-result URLs. Institutional websites are not visited. Queries go only to registry services; complete responses and source provenance remain in the private development corpus. Services without IANA discovery, timeouts, access limits, empty replies, syntax errors, and parsed registration or policy results are tracked separately. The inventory records verified service behavior, parsing coverage, and the outcomes of bounded registry probes.

| Namespace or response family | Handling |
| --- | --- |
| `.ac.uk` | Query the [Jisc namespace registry](https://www.jisc.ac.uk/domain-registry?lang=en) through `whois.ja.net:43`; normalize owner, registration agent, servers, and ordinal entry/renewal dates. |
| `.gov.uk` | Query [Nominet's separate government registry](https://registrars.nominet.uk/registry/dot-gov-uk/) through its dedicated HTTPS RDAP route. |
| `.nhs.uk`, `.police.uk`, `.mod.uk`, `.mil.uk`, `.parliament.uk` children | A 404 from Nominet's parent `.uk` interface means the parent registry did not provide independent registration data for the queried child. Successful disclosed records remain visible. These [public-sector namespaces](https://www.gov.uk/guidance/check-if-your-organisation-can-get-a-govuk-domain-name) have distinct administration; a parent-registry absence does not determine child availability. |
| Parent-only RDAP / WHOIS objects | Report `Registry did not provide domain-specific registration data` when the object identifies a known public namespace above the queried child; discard the parent's contact and delegation data. This also covers the observed RNIDS `.mil.rs` delegation response. |
| `.ac.jp`, `.ed.jp`, `.go.jp`, `.lg.jp` category names | Preserve JPRS's explicit Reserved result, including category records with delegation nameservers. Child registrations retain their own disclosed fields. [JPRS defines the eligible organization types](https://jprs.jp/about/jp-dom/spec/). |
| `.sn` HTTPS WHOIS envelope | Decode the registry's single-field JSON wrapper locally, normalize French domain and holder labels, and preserve contact/date scope. No second query is made. |
| Exact `.ie`, `.nz`, `.sx` usage-restriction replies | Report the registry's restriction without inventing an owner or availability result. |
| Eligibility-controlled categories with ordinary records | Parse the disclosed record normally. For example, [Registro.br's category rules](https://registro.br/dominio/categorias) include education, government, military, judicial, legislative, and professional uses; [auDA](https://www.auda.org.au/au-domain-names/the-different-au-domain-names/) defines education and government categories. Eligibility rules alone do not change a lookup result. |

Other services continue through ordinary IANA discovery and shared parsing. Fixed namespace-registry routes are selected before querying; registrar referrals never provide destinations. Unrelated identities and contradictory records remain errors. A direct query for a namespace, such as `ac.uk` or `gov.rs`, asks for that namespace's own record.

## WHOIS-only suffixes

The following tables include every WHOIS-only suffix in the snapshot, including the explicit `.tw` classification. Shared parsing and observed registry-family rules supply the standard layout.

### Ordinary ASCII TLDs

| Registry WHOIS hostname | Suffixes |
| --- | --- |
| `kero.yachay.pe` | `.pe` |
| `whois.aeda.net.ae` | `.ae` |
| `whois.afilias-srs.net` | `.pr` |
| `whois.amnic.net` | `.am` |
| `whois.ande.gov.gn` | `.gn` |
| `whois.ati.tn` | `.tn` |
| `whois.ax` | `.ax` |
| `whois.bnnic.bn` | `.bn` |
| `whois.cctld.by` | `.by` |
| `whois.cmc.iq` | `.iq` |
| `whois.cnnic.cn` | `.cn` |
| `whois.co.ug` | `.ug` |
| `whois.denic.de` | `.de` |
| `whois.dmdomains.dm` | `.dm` |
| `whois.dnrs.vu` | `.vu` |
| `whois.dns.be` | `.be` |
| `whois.dns.hr` | `.hr` |
| `whois.dns.lu` | `.lu` |
| `whois.dns.pt` | `.pt` |
| `whois.dominio.gq` | `.gq` |
| `whois.domreg.lt` | `.lt` |
| `whois.dot.cf` | `.cf` |
| `whois.dot.tk` | `.tk` |
| `whois.educause.edu` | `.edu` |
| `whois.eu` | `.eu` |
| `whois.gg` | `.gg` |
| `whois.hkirc.hk` | `.hk` |
| `whois.iana.org` | `.arpa` |
| `whois.identitydigital.services` | `.gi`, `.vc` |
| `whois.iis.nu` | `.nu` |
| `whois.iis.se` | `.se` |
| `whois.irs.net.nz` | `.nz` |
| `whois.isoc.org.il` | `.il` |
| `whois.je` | `.je` |
| `whois.jprs.jp` | `.jp` |
| `whois.marnet.mk` | `.mk` |
| `whois.mediaserv.net` | `.gf`, `.mq` |
| `whois.monic.mo` | `.mo` |
| `whois.mx` | `.mx` |
| `whois.mynic.my` | `.my` |
| `whois.nc` | `.nc` |
| `whois.nic.ac` | `.ac` |
| `whois.nic.af` | `.af` |
| `whois.nic.ag` | `.ag` |
| `whois.nic.at` | `.at` |
| `whois.nic.aw` | `.aw` |
| `whois.nic.bj` | `.bj` |
| `whois.nic.bo` | `.bo` |
| `whois.nic.ch` | `.ch` |
| `whois.nic.ci` | `.ci` |
| `whois.nic.cl` | `.cl` |
| `whois.nic.do` | `.do` |
| `whois.nic.dz` | `.dz` |
| `whois.nic.es` | `.es` |
| `whois.nic.ge` | `.ge` |
| `whois.nic.gh` | `.gh` |
| `whois.nic.gl` | `.gl` |
| `whois.nic.gp` | `.gp` |
| `whois.nic.hu` | `.hu` |
| `whois.nic.im` | `.im` |
| `whois.nic.io` | `.io` |
| `whois.nic.ir` | `.ir` |
| `whois.nic.it` | `.it` |
| `whois.nic.ki` | `.ki` |
| `whois.nic.kn` | `.kn` |
| `whois.nic.kz` | `.kz` |
| `whois.nic.la` | `.la` |
| `whois.nic.li` | `.li` |
| `whois.nic.ls` | `.ls` |
| `whois.nic.lv` | `.lv` |
| `whois.nic.mc` | `.mc` |
| `whois.nic.md` | `.md` |
| `whois.nic.me` | `.me` |
| `whois.nic.mn` | `.mn` |
| `whois.nic.mr` | `.mr` |
| `whois.nic.mw` | `.mw` |
| `whois.nic.mz` | `.mz` |
| `whois.nic.net.bw` | `.bw` |
| `whois.nic.net.jo` | `.jo` |
| `whois.nic.net.sa` | `.sa` |
| `whois.nic.net.sb` | `.sb` |
| `whois.nic.org.uy` | `.uy` |
| `whois.nic.sc` | `.sc` |
| `whois.nic.sh` | `.sh` |
| `whois.nic.sm` | `.sm` |
| `whois.nic.so` | `.so` |
| `whois.nic.st` | `.st` |
| `whois.nic.tc` | `.tc` |
| `whois.nic.td` | `.td` |
| `whois.nic.tg` | `.tg` |
| `whois.nic.tl` | `.tl` |
| `whois.nic.tm` | `.tm` |
| `whois.nic.us` | `.us` |
| `whois.nic.ve` | `.ve` |
| `whois.pknic.net.pk` | `.pk` |
| `whois.punktum.dk` | `.dk` |
| `whois.register.bg` | `.bg` |
| `whois.registre.bf` | `.bf` |
| `whois.registre.ma` | `.ma` |
| `whois.registry.co` | `.co` |
| `whois.registry.gov.mm` | `.mm` |
| `whois.registry.hm` | `.hm` |
| `whois.registry.om` | `.om` |
| `whois.registry.pf` | `.pf` |
| `whois.registry.qa` | `.qa` |
| `whois.rnids.rs` | `.rs` |
| `whois.rotld.ro` | `.ro` |
| `whois.sk-nic.sk` | `.sk` |
| `whois.sx` | `.sx` |
| `whois.tcinet.ru` | `.ru`, `.su` |
| `whois.tld.ee` | `.ee` |
| `whois.tld.sy` | `.sy` |
| `whois.trabis.gov.tr` | `.tr` |
| `whois.twnic.net.tw` | `.tw` |
| `whois.weare.ie` | `.ie` |
| `whois.website.ws` | `.ws` |
| `whois1.nic.bi` | `.bi` |

### Internationalized TLDs

Unicode names are followed by their ASCII A-label identities. Queries use the ASCII form.

| Registry WHOIS hostname | Suffixes |
| --- | --- |
| `whois.aeda.net.ae` | `.امارات` (`.xn--mgbaam7a8h`) |
| `whois.amnic.net` | `.հայ` (`.xn--y9a3aq`) |
| `whois.ati.tn` | `.تونس` (`.xn--pgbs0dh`) |
| `whois.cctld.by` | `.бел` (`.xn--90ais`) |
| `whois.cmc.iq` | `.عراق` (`.xn--mgbtx2b`) |
| `whois.cnnic.cn` | `.中国` (`.xn--fiqs8s`), `.中國` (`.xn--fiqz9s`) |
| `whois.dotukr.com` | `.укр` (`.xn--j1amh`) |
| `whois.eu` | `.ею` (`.xn--e1a4c`), `.ευ` (`.xn--qxa6a`) |
| `whois.hkirc.hk` | `.香港` (`.xn--j6w193g`) |
| `whois.imena.bg` | `.бг` (`.xn--90ae`) |
| `whois.isoc.org.il` | `.ישראל` (`.xn--4dbrk0ce`) |
| `whois.marnet.mk` | `.мкд` (`.xn--d1alf`) |
| `whois.mn` | `.мон` (`.xn--l1acc`) |
| `whois.monic.mo` | `.澳門` (`.xn--mix891f`) |
| `whois.mynic.my` | `.مليسيا` (`.xn--mgbx4cd0ab`) |
| `whois.nic.dz` | `.الجزائر` (`.xn--lgbbat1ad8j`) |
| `whois.nic.ir` | `.ایران` (`.xn--mgba3a4f16a`) |
| `whois.nic.kz` | `.қаз` (`.xn--80ao21a`) |
| `whois.nic.la` | `.ລາວ` (`.xn--q7ce6a`) |
| `whois.nic.mr` | `.موريتانيا` (`.xn--mgbah1a3hjkrd`) |
| `whois.nic.net.sa` | `.السعودية` (`.xn--mgberp4a5d4ar`) |
| `whois.nixiregistry.in` | `.ಭಾರತ` (`.xn--2scrj9c`), `.ଭାରତ` (`.xn--3hcrj9c`), `.ভাৰত` (`.xn--45br5cyl`), `.ভারত` (`.xn--45brj9c`), `.భారత్` (`.xn--fpcrj9c3d`), `.ભારત` (`.xn--gecrj9c`), `.भारतम्` (`.xn--h2breg3eve`), `.भारत` (`.xn--h2brj9c`), `.भारोत` (`.xn--h2brj9c8c`), `.بارت` (`.xn--mgbbh1a`), `.بھارت` (`.xn--mgbbh1a71e`), `.ڀارت` (`.xn--mgbgu82a`), `.ഭാരതം` (`.xn--rvc1e0am3e`), `.ਭਾਰਤ` (`.xn--s9brj9c`), `.இந்தியா` (`.xn--xkc2dl3a5ee0h`) |
| `whois.pnina.ps` | `.فلسطين` (`.xn--ygbi2ammx`) |
| `whois.registry.om` | `.عمان` (`.xn--mgb9awbf`) |
| `whois.registry.qa` | `.قطر` (`.xn--wgbl6a`) |
| `whois.rnids.rs` | `.срб` (`.xn--90a3ac`) |
| `whois.tcinet.ru` | `.рф` (`.xn--p1ai`) |
| `whois.tld.sy` | `.سورية` (`.xn--ogbpf8fl`) |
| `whois.twnic.net.tw` | `.台湾` (`.xn--kprw13d`) |

## Verification scope

Verification inventories both WHOIS-only suffixes and the registry WHOIS services published for RDAP-primary ccTLDs. It includes ordinary registrations, registry-operated names with exceptional records, education/government and other special namespaces, IDN suffixes, and explicit negative candidates. Service responses, parsed records, access limits, and unreachable endpoints remain separate outcomes.

Contact scope, date precision, reserved or restricted results, IDN identities, DNSSEC, and protocol-fallback boundaries receive synthetic regression coverage. Developer response bodies remain under ignored `test/whois` and `test/rdap`; runtime does not retain queries or responses.

## Suffixes without an advertised service

The following 70 active TLDs publish neither protocol in this snapshot. Domain Check reports a service-discovery error.

`.al`, `.ao`, `.aq`, `.az`, `.ba`, `.bb`, `.bd`, `.bs`, `.bt`, `.bv`

`.bz`, `.cd`, `.cg`, `.ck`, `.cu`, `.cw`, `.cy`, `.dj`, `.eg`, `.er`

`.et`, `.fk`, `.ga`, `.gb`, `.gm`, `.gr`, `.gt`, `.gu`, `.gw`, `.jm`

`.kh`, `.km`, `.kp`, `.kw`, `.lc`, `.lk`, `.lr`, `.mh`, `.mil`, `.mp`

`.mt`, `.mv`, `.ne`, `.ni`, `.np`, `.nr`, `.pa`, `.ph`, `.ps`, `.py`

`.sj`, `.sl`, `.sv`, `.sz`, `.tj`, `.tt`, `.va`, `.vn`, `.বাংলা` (`.xn--54b7fta0cc`), `.ලංකා` (`.xn--fzc2c9e2c`)

`.پاکستان` (`.xn--mgbai9azgqp6j`), `.الاردن` (`.xn--mgbayh7gpa`), `.المغرب` (`.xn--mgbc0a9azcg`), `.سودان` (`.xn--mgbpl2fh`), `.გე` (`.xn--node`), `.ελ` (`.xn--qxam`), `.مصر` (`.xn--wgbh1c`), `.இலங்கை` (`.xn--xkc2al3hye2a`), `.za`, `.zw`

## Interpreting a query

- `domain example.com` queries that exact normalized domain. A URL such as `https://www.example.com/path` is reduced locally to `example.com`.
- Internationalized and validated emoji labels are converted to ASCII/Punycode for querying. Each registry determines registration eligibility.
- Multi-label namespaces and reserved names are handled through the registry's explicit responses. Registration records, policy restrictions, and service errors have distinct results.
- Domain Check displays the registrant details disclosed by the registry and omits undisclosed rows, keeping the lookup at the source.
- Service access restrictions are reported clearly. For the education-registry fallback described in the privacy policy, public nameserver records provide delegation information.

See [technical.md](technical.md) for routing and parsing, [security.md](security.md) for security controls, and [quality-assurance.md](quality-assurance.md) for validation requirements.
