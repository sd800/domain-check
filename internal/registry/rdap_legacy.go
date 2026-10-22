package registry

import (
	"encoding/json"
	"strings"
)

// NIC Senegal's IANA-listed HTTPS interface wraps its WHOIS text in JSON.
// Accept only that single-field envelope for .sn, through the same bounded,
// identity-checked parser. This performs no secondary WHOIS request.
func parseSNEnvelope(domain string, data []byte) (Record, error) {
	var envelope map[string]json.RawMessage
	if err := unmarshalResponse(data, &envelope); err != nil || len(envelope) != 1 {
		return Record{}, queryError("invalid-response", "invalid registry WHOIS envelope")
	}
	var response string
	if json.Unmarshal(envelope["whois43response"], &response) != nil || response == "" {
		return Record{}, queryError("invalid-response", "invalid registry WHOIS envelope")
	}
	return parseWHOIS(domain, strings.ReplaceAll(response, `\r\n`, "\n"))
}
