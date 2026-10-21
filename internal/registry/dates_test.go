package registry

import "testing"

func TestNormalizedDateFormatsAndUTCSuffix(t *testing.T) {
	for input, want := range map[string]string{
		"23-May-1985":               "1985-05-23",
		"03-MAY-1985":               "1985-05-03",
		"3-May-1985":                "1985-05-03",
		"1985-5-3":                  "1985-05-03",
		"1985/5/3":                  "1985-05-03",
		"1985.05.03":                "1985-05-03",
		"3 May 1985":                "1985-05-03",
		"March 3, 1985":             "1985-03-03",
		"2003-03-10T11:19:18":       "2003-03-10 11:19:18",
		"23-May-1985 11:19:18":      "1985-05-23 11:19:18",
		"2003-03-10 11:19:18":       "2003-03-10 11:19:18",
		"2003-03-10T11:19:18+08:00": "2003-03-10 03:19:18 UTC",
		"2003-03-10T11:19:18+0800":  "2003-03-10 03:19:18 UTC",
		"2026-01-01T00:30:00+08:00": "2025-12-31 16:30:00 UTC",
		"2025-05-15T10:25:25Z":      "2025-05-15 10:25:25 UTC",
		"2003-03-10 11:19:18 UTC":   "2003-03-10 11:19:18 UTC",
		"23-May-1985 11:19:18 GMT":  "1985-05-23 11:19:18 UTC",
		"2003-03-10 11:19:18 CST":   "2003-03-10 11:19:18 CST",
		"before Aug-1996":           "before Aug-1996",
		"31-Feb-2026":               "31-Feb-2026",
		"03/04/2026":                "03/04/2026",
		"n/a":                       "",
	} {
		t.Run(input, func(t *testing.T) {
			if got := formatTime(input); got != want {
				t.Fatalf("got %q; want %q", got, want)
			}
		})
	}
}

func TestWHOISDateOnlyFields(t *testing.T) {
	for _, test := range []struct {
		name, response, registration, expiration, changed string
	}{
		{
			name: "EDUCAUSE registry dates",
			response: "Domain Name: MIT.EDU\n" +
				"Domain record activated:    23-May-1985\n" +
				"Domain record last updated: 25-Jun-2026\n" +
				"Domain expires:             31-Jul-2027\n",
			registration: "1985-05-23", expiration: "2027-07-31", changed: "2026-06-25",
		},
		{
			name: "ISO dates in ordinary fields",
			response: "Domain Name: mit.edu\n" +
				"Creation Date: 1985-05-23\n" +
				"Updated Date: 2026-06-25\n" +
				"Registry Expiry Date: 2027-07-31\n",
			registration: "1985-05-23", expiration: "2027-07-31", changed: "2026-06-25",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, err := parseWHOIS("mit.edu", test.response)
			if err != nil || r.Registration != test.registration || r.Expiration != test.expiration || r.Changed != test.changed {
				t.Fatalf("date-only values were lost or incorrectly formatted: %+v, %v", r, err)
			}
		})
	}
}

func TestRDAPDateOnlyEvents(t *testing.T) {
	r, err := parseRDAP("mit.edu", []byte(`{
		"objectClassName":"domain","ldhName":"mit.edu",
		"events":[
			{"eventAction":"registration","eventDate":"1985-05-23"},
			{"eventAction":"expiration","eventDate":"2027-07-31"},
			{"eventAction":"last changed","eventDate":"2026-06-25"},
			{"eventAction":"last update of RDAP database","eventDate":"2026-10-21"}
		]
	}`))
	if err != nil || r.Registration != "1985-05-23" || r.Expiration != "2027-07-31" || r.Changed != "2026-06-25" {
		t.Fatalf("date-only events were lost or changed: %+v, %v", r, err)
	}
}

func TestRegistrySpecificWHOISDates(t *testing.T) {
	for _, test := range []struct{ tld, input, want string }{
		{"pt", "21/10/2000 01:02:03", "2000-10-21 01:02:03"},
		{"rs", "21.10.2000 01:02:03", "2000-10-21 01:02:03"},
		{"hk", "21-10-2000", "2000-10-21"},
		{"be", "Sat Oct 21 2000", "2000-10-21"},
		{"jo", "Oct 21 2000 1:02PM", "2000-10-21 13:02:00"},
		{"at", "20001021 01:02:03", "2000-10-21 01:02:03"},
		{"jp", "2000/10/21 01:02:03 (JST)", "2000-10-20 16:02:03 UTC"},
		{"tn", "21-10-2000 01:02:03 GMT+1", "2000-10-21 00:02:03 UTC"},
		{"kz", "2000-10-21 01:02:03 (GMT+0:00)", "2000-10-21 01:02:03 UTC"},
		{"gg", "21st October 2000 at 01:02:03.000", "2000-10-21 01:02:03"},
		{"tr", "2000-Oct-21.", "2000-10-21"},
		{"cl", "2000-10-21 01:02:03 CLST", "2000-10-21 04:02:03 UTC"},
		{"cl", "2000-10-21 01:02:03 CLT", "2000-10-21 05:02:03 UTC"},
		{"tm", "Never", "Never"},
		{"ro", "Before 2000", "Before 2000"},
		{"cn", "03/04/2000", "03/04/2000"},
		{"cn", "2000-10-21 01:02:03 CST", "2000-10-21 01:02:03 CST"},
	} {
		t.Run(test.tld+" "+test.input, func(t *testing.T) {
			if got := formatWHOISTime(test.tld, test.input); got != test.want {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}
