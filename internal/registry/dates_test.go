package registry

import "testing"

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
			registration: "23-May-1985", expiration: "31-Jul-2027", changed: "25-Jun-2026",
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
				t.Fatalf("date-only values were lost or changed: %+v, %v", r, err)
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
