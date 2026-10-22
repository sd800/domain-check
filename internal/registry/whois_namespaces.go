package registry

import "strings"

// Normalize exact labels from observed namespace formats before the shared
// scope parser. Contact dates remain contact dates; no field triggers a lookup.
func tuneNamespaceWHOIS(domain string, lines []whoisLine) []whoisLine {
	jisc, senegal := strings.HasSuffix(domain, ".ac.uk"), strings.HasSuffix(domain, ".sn")
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	korean := tld == "kr" || tld == "xn--3e0b707e"
	if tld == "kg" {
		lines = tokenizeKyrgyzHeader(lines)
	}
	if !jisc && !senegal && !korean && tld != "kg" && tld != "is" && tld != "br" {
		return lines
	}
	for i := range lines {
		line := &lines[i]
		if korean && (strings.TrimSpace(line.raw) == "1차 네임서버 정보" || strings.TrimSpace(line.raw) == "2차 네임서버 정보") {
			line.key, line.value, line.labeled = "nameservers", "", true
		}
		if !line.labeled {
			continue
		}
		if jisc {
			switch line.key {
			case "domain owner":
				line.key = "registrant"
			case "registered by":
				line.key = "registrar"
			case "registrant contact":
				line.key = "administrative contact"
			case "servers":
				line.key = "nameservers"
			case "entry created":
				line.key = "domain created"
			case "entry updated":
				line.key = "domain last updated date"
			case "renewal date":
				line.key = "domain expiration date"
			}
		}
		if senegal {
			switch line.key {
			case "nom de domaine":
				line.key = "domain name"
			case "date de création":
				line.key = "creation date"
			case "date d'expiration":
				line.key = "expiration date"
			case "dernière modification":
				line.key = "last modified"
			case "statut":
				line.key = "status"
			case "serveur de noms":
				line.key = "name server"
			case "nom":
				line.key = "name"
			case "courriel":
				line.key = "email"
			case "adresse":
				line.key = "address"
			case "pays":
				line.key = "country"
			case "billing c":
				line.key = "billing contact"
			}
		}
		if tld == "kg" {
			switch line.key {
			case "record last updated on":
				line.key = "domain last updated date"
			case "name servers in the listed order":
				line.key = "nameservers"
			}
		}
		if tld == "is" && line.key == "role" {
			line.key = "organization"
		}
		if tld == "br" && line.key == "nic hdl br" {
			line.key = "nic hdl"
		}
		if korean {
			switch line.key {
			case "도메인이름":
				line.key = "domain name"
			case "등록인":
				line.key = "registrant"
			case "책임자", "administrative contact(ac)":
				line.key = "administrative contact"
			case "등록일", "registered date":
				line.key = "domain registration date"
			case "최근 정보 변경일", "last updated date":
				line.key = "domain last updated date"
			case "사용 종료일", "expiration date":
				line.key = "domain expiration date"
			case "호스트이름":
				line.key = "name server"
			case "등록대행자", "authorized agency":
				line.key = "registrar name"
				if index := strings.Index(line.value, "(http"); index >= 0 {
					line.value = strings.TrimSpace(line.value[:index])
				}
			}
		}
		if whoisKeys[line.key] == "domain" {
			if _, value, ok := strings.Cut(line.raw, ":"); ok {
				line.value = whoisFieldValue(line.key, value)
			}
		}
	}
	return lines
}

func namespaceUsageRestriction(lines []whoisLine) bool {
	code, message := false, false
	for _, line := range lines {
		if strings.TrimSpace(line.raw) == "" || strings.HasPrefix(strings.TrimSpace(line.raw), "%") {
			continue
		}
		switch {
		case line.key == "error code" && line.value == "01044":
			code = true
		case line.key == "error message" && line.value == "The domain name requested has usage restrictions applied to it. Please see your Registrar for more details.":
			message = true
		default:
			return false
		}
	}
	return code && message
}
