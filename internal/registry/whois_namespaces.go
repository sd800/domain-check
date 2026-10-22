package registry

import "strings"

// Normalize exact labels from observed namespace formats before the shared
// scope parser. Contact dates remain contact dates; no field triggers a lookup.
func tuneNamespaceWHOIS(domain string, lines []whoisLine) {
	jisc, senegal := strings.HasSuffix(domain, ".ac.uk"), strings.HasSuffix(domain, ".sn")
	if !jisc && !senegal {
		return
	}
	for i := range lines {
		line := &lines[i]
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
		if whoisKeys[line.key] == "domain" {
			if _, value, ok := strings.Cut(line.raw, ":"); ok {
				line.value = whoisFieldValue(line.key, value)
			}
		}
	}
}

func namespaceUsageRestriction(domain string, lines []whoisLine) bool {
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	if tld != "ie" && tld != "nz" && tld != "sx" {
		return false
	}
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
