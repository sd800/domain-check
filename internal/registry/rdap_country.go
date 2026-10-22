package registry

import (
	"encoding/json"
	"strings"
)

// Country codes are supplied by the selected entity, never inferred from its name,
// another contact, a label, or an external service. RFC 8605's cc takes precedence.
func contactCountry(parameters json.RawMessage, value any) string {
	var params map[string]json.RawMessage
	if json.Unmarshal(parameters, &params) != nil {
		return ""
	}
	explicit, country := false, ""
	for key, raw := range params {
		if strings.EqualFold(key, "cc") {
			var code string
			if json.Unmarshal(raw, &code) != nil {
				return ""
			}
			code = countryCode(code)
			if code == "" || explicit && code != country {
				return ""
			}
			explicit, country = true, code
		}
	}
	if explicit {
		return country
	}
	if parts, ok := value.([]any); ok && len(parts) == 7 {
		if country, ok := parts[6].(string); ok {
			return countryCode(country)
		}
	}
	return ""
}

func countryCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) == 2 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z' {
		return value
	}
	return ""
}
