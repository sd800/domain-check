package registry

import "strings"

// TWNIC can answer a delegated namespace rather than the queried child. This
// discloses no child registration or availability; referrals remain unfetched.
func twnicNamespaceOnly(domain string, lines []whoisLine) bool {
	if strings.HasSuffix(domain, ".mil.tw") {
		var text []string
		for _, line := range lines {
			if value := strings.TrimSpace(line.raw); value != "" {
				text = append(text, value)
			}
		}
		return len(text) == 1 && text[0] == "網域名稱不合規定"
	}
	if strings.HasSuffix(domain, ".edu.tw") {
		parent := false
		for _, line := range lines {
			if whoisKeys[line.key] != "domain" {
				continue
			}
			name, err := normalizeWHOISDomain(line.value)
			if err != nil || name != "edu.tw" {
				return false
			}
			parent = true
		}
		return parent
	}
	if strings.HasSuffix(domain, ".gov.tw") {
		referral := false
		for _, line := range lines {
			if strings.TrimSpace(line.raw) == "" {
				continue
			}
			if line.key != "referralserver" || line.value != "rwhois://rs.gsn.gov.tw" {
				return false
			}
			referral = true
		}
		return referral
	}
	return false
}
