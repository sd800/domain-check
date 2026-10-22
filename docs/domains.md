# Domain suffixes and registry coverage

Domain Check accepts domain names and full URLs, including internationalized labels. Routing is discovered from the current IANA datasets; this document is a dated service inventory, not a hardcoded allowlist or a promise that every registry is reachable.

## IANA snapshot

Source: the complete [IANA TLD Registry Data](https://data.iana.org/TLD/v1/tlds.json), generated **2026-10-21T21:54:03+00:00**, and [IANA RDAP DNS bootstrap data](https://data.iana.org/rdap/dns.json), published **2026-09-30T23:00:03Z**. The TLD registry's RDAP fields do not replace the DNS bootstrap as the RDAP routing source.

| Active top-level suffixes | Count |
| --- | ---: |
| RDAP advertised in `dns.json` | 1203 |
| No RDAP advertisement, WHOIS hostname published | 164 |
| Neither RDAP nor a WHOIS hostname published | 70 |
| Total active TLDs in this snapshot | 1437 |

WHOIS-only coverage includes 119 ordinary ASCII TLDs and 45 internationalized TLDs. IDN classification refers to the top-level suffix itself; a Unicode label under `.com` does not make `.com` an IDN TLD.

Run `domain update` to refresh the actual routing datasets. Published services can change after this snapshot. A listed server does not establish current accessibility, successful parsing of every format, registration eligibility, or ownership disclosure.

## RDAP suffixes

The full current RDAP list and endpoint assignments are available in the [official DNS bootstrap](https://data.iana.org/rdap/dns.json). Domain Check selects the longest matching suffix and prefers HTTPS. It never follows registrar referrals or switches to WHOIS after an advertised RDAP service fails.

## WHOIS-only suffixes

The following tables include every active WHOIS-only suffix in the snapshot, grouped by its published registry hostname. Shared field parsing applies to common formats; dedicated format rules are used only for observed differences. Absence of a dedicated suffix flag does not imply absence of parsing support.

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

The registry-format audit corresponding to this snapshot retained 653 developer response samples. Replies were obtained for 149 of the 164 WHOIS-only suffixes; 137 suffixes supplied at least one positive parsed record. The remaining cases include explicit negatives, policy or service responses, and unreachable endpoints. These are observations from a bounded audit, not uptime measurements or universal compatibility guarantees.

Coverage checks include non-country namespaces such as `.edu` and `.arpa`, registry-operated names with minimal records, ordinary benign registered domains, and explicit negative candidates. Contact scope, date precision, IDN identities, DNSSEC, and service failures are checked separately. Real response bodies and registrant details remain outside the public repository; public fixtures are synthetic.

## Suffixes without an advertised service

The following 70 active TLDs had neither an RDAP entry nor a WHOIS hostname in the snapshot. Domain Check reports the absence of a public registry service rather than guessing an endpoint or contacting a commercial lookup site.

`.al`, `.ao`, `.aq`, `.az`, `.ba`, `.bb`, `.bd`, `.bs`, `.bt`, `.bv`

`.bz`, `.cd`, `.cg`, `.ck`, `.cu`, `.cw`, `.cy`, `.dj`, `.eg`, `.er`

`.et`, `.fk`, `.ga`, `.gb`, `.gm`, `.gr`, `.gt`, `.gu`, `.gw`, `.jm`

`.kh`, `.km`, `.kp`, `.kw`, `.lc`, `.lk`, `.lr`, `.mh`, `.mil`, `.mp`

`.mt`, `.mv`, `.ne`, `.ni`, `.np`, `.nr`, `.pa`, `.ph`, `.ps`, `.py`

`.sj`, `.sl`, `.sv`, `.sz`, `.tj`, `.tt`, `.va`, `.vn`, `.বাংলা` (`.xn--54b7fta0cc`), `.ලංකා` (`.xn--fzc2c9e2c`)

`.پاکستان` (`.xn--mgbai9azgqp6j`), `.الاردن` (`.xn--mgbayh7gpa`), `.المغرب` (`.xn--mgbc0a9azcg`), `.سودان` (`.xn--mgbpl2fh`), `.გე` (`.xn--node`), `.ελ` (`.xn--qxam`), `.مصر` (`.xn--wgbh1c`), `.இலங்கை` (`.xn--xkc2al3hye2a`), `.za`, `.zw`

## Interpreting a query

- `domain example.com` queries that exact normalized domain. A URL such as `https://www.example.com/path` is reduced locally to `example.com`.
- Internationalized and validated emoji labels are converted to ASCII/Punycode. Successful conversion does not establish that the registry allows the name.
- Multi-label namespaces and reserved names can return explicit registry restrictions. Missing fields do not mean a domain is unregistered, and DNS absence does not establish availability.
- A registry may withhold registrant names or emails. Domain Check omits those rows and never queries a registrar to fill them.
- Some services impose geographic or institutional access restrictions. Transport failures remain errors; the disclosed education-registry DNS fallback reports delegation only.

See [technical.md](technical.md) for routing and parsing, [security.md](security.md) for trust boundaries, and [quality-assurance.md](quality-assurance.md) for validation requirements.
