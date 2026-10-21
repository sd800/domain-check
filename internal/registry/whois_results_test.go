package registry

import "testing"

func TestWHOISNegativeStatusFields(t *testing.T) {
	for _, test := range []struct {
		status string
		want   Result
	}{
		{"AVAILABLE", NotRegistered},
		{"free", NotRegistered},
		{"Not Registered, and may be available if valid", NotRegistered},
		{"No Object Found", NoRecord},
		{"Reserved", Reserved},
		{"Prohibited String - Domain Cannot Be Registered", NotAllowed},
	} {
		for _, label := range []string{"Status", "Domain Status", "Registration Status"} {
			r, err := parseWHOIS("example.test", "Domain Name: example.test\n"+label+":\t\t"+test.status+"\n")
			if err != nil || r.Result != test.want || len(r.Statuses) != 0 {
				t.Errorf("%s: %s: %+v, %v", label, test.status, r, err)
			}
		}
	}
	r, err := parseWHOIS("example.test", "Domain Name: example.test\nRegistrant:\nName: Example Owner\nStatus: available\n")
	if err != nil || r.Result != "" || r.Registrant != "Example Owner" {
		t.Fatalf("contact status became domain availability: %+v, %v", r, err)
	}
	if _, err := parseWHOIS("example.test", "Domain Name: example.test\nCreated: 2000-10-21\nStatus: available"); err == nil {
		t.Fatal("accepted contradictory registration and availability data")
	}
}

func TestWHOISNegativeReplyFormats(t *testing.T) {
	for _, body := range []string{
		"No match", "No match!!", "% nothing found", "*** Nothing found for this query.",
		"object does not exist", "No_Se_Encontro_El_Objeto/Object_Not_Found",
		"example.test: no entries found.", "example.test - No Match",
		`domain "example.test" not found.`, "The domain example.test was not found.",
		"No record found for 'example.test'.", "No match found for example.test",
		"No Match for domain: example.test", `% No match for domain 'example.test'.`,
	} {
		r, err := parseWHOIS("example.test", body)
		if err != nil || r.Result != NoRecord {
			t.Errorf("%q: %+v, %v", body, r, err)
		}
	}
	for _, body := range []string{"The domain has not been registered.", "Domain example.test is available for purchase", ">>> Domain example.test is available for registration\n>>> Please visit https://registrar.test/\nAll rights\nreserved."} {
		r, err := parseWHOIS("example.test", body)
		if err != nil || r.Result != NotRegistered {
			t.Errorf("%q: %+v, %v", body, r, err)
		}
	}
	for _, body := range []string{
		`domain "other.test" not found.`, "The domain other.test was not found.",
		"No record found for 'other.test'.", "No Match for domain: other.test",
		"other.test: no entries found.", "other.test - No Match",
		"Domain other.test is available for purchase", "No match for a different query.",
		"% Terms: a missing record does not guarantee availability.", "Malformed request.",
		"% Terms of use only, without a query result.",
		"All rights\nreserved.", "Domain other.test is reserved",
	} {
		if r, err := parseWHOIS("example.test", body); err == nil || r.Result != "" {
			t.Errorf("unrelated or ambiguous reply became availability: %q: %+v, %v", body, r, err)
		}
	}
}

func TestWHOISNegativeFooterCannotReplaceResult(t *testing.T) {
	for _, body := range []string{
		"No match!!\n>>> Last update of WHOIS database: 2026-10-21T00:00:00Z <<<\nAll rights\nreserved.",
		"No matching record.\nTerms of Use:\nThis name is reserved by the Registry.",
		"No entries found\nCopyright notice:\nReserved Domain Name",
	} {
		r, err := parseWHOIS("example.test", body)
		if err != nil || r.Result != NoRecord {
			t.Fatalf("footer replaced a missing-record result: %+v, %v", r, err)
		}
	}
}

func TestWHOISRegistryPolicyRejectionIsNotAnEmptyRecord(t *testing.T) {
	r, err := parseWHOIS("example.test", `Domain Name: example.test
>>> The domain contains special characters not allowed.
>>> This domain violates registry policy.
>>> Last update of WHOIS database: 2026-10-21T00:00:00Z <<<`)
	if err != nil || r.Result != NotAllowed || r.Registration != "" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestCNNICNamespaceAndRestrictedNames(t *testing.T) {
	// The registry returns the same statement for namespace labels and
	// restricted names. It is not evidence of ordinary availability or a
	// permanent reservation, and its registrar suggestion is never followed.
	for _, domain := range []string{"net.cn", "home.cn", "example.cn", "example.net.cn"} {
		r, err := parseWHOIS(domain, "the Domain Name you apply can not be registered online. Please consult your Domain Name registrar\n")
		if err != nil || r.Domain != domain || r.Result != Restricted || r.Registrar != "" {
			t.Fatalf("%s: %+v, %v", domain, r, err)
		}
	}
	r, err := parseWHOIS("example.net.cn", "No matching record.\n")
	if err != nil || r.Result != NoRecord {
		t.Fatalf("ordinary missing child domain became a restriction: %+v, %v", r, err)
	}
}
