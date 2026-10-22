package registry

// Profiles describe only observed format differences. Field aliases and
// contact/object interpretation remain shared across registry responses.
type whoisProfile uint32

const (
	dayFirstDates whoisProfile = 1 << iota
	ordinalDates
	periodDateSuffix
	chileanTimezones
	changeHistoryDates
	registrantHandle
	holderHandle
	domainContactEmail
	domainDatesAfterContact
	jprsContactFields
	nameserverObjectDates
	implicitDomainIdentity
	domainDNSSECAfterContact
	monthNameDates
	ianaDomainMetadata
	cnnicVariantNames
	twnicVariantNames
	hkBundledNames
	lithuanianContacts
	luxembourgStatus
	ugandanDomainUpdate
	bulgarianStatus
	multilineRegistrant
	twnicContactFields
)

func (p whoisProfile) has(flag whoisProfile) bool { return p&flag != 0 }

var whoisProfiles = map[string]whoisProfile{
	"am":              domainDatesAfterContact,
	"arpa":            ianaDomainMetadata,
	"at":              registrantHandle,
	"ax":              dayFirstDates,
	"bg":              bulgarianStatus,
	"by":              domainContactEmail,
	"cl":              chileanTimezones | implicitDomainIdentity,
	"edu":             monthNameDates | multilineRegistrant,
	"ee":              nameserverObjectDates,
	"gg":              ordinalDates,
	"hk":              dayFirstDates,
	"il":              dayFirstDates | changeHistoryDates | domainContactEmail,
	"im":              dayFirstDates,
	"ir":              registrantHandle,
	"je":              ordinalDates,
	"jp":              jprsContactFields,
	"ls":              dayFirstDates | registrantHandle,
	"lt":              lithuanianContacts,
	"lu":              luxembourgStatus,
	"mk":              dayFirstDates | registrantHandle,
	"mw":              dayFirstDates | registrantHandle,
	"nu":              holderHandle,
	"pt":              dayFirstDates,
	"rs":              dayFirstDates,
	"se":              holderHandle,
	"sm":              dayFirstDates,
	"tn":              dayFirstDates | domainDNSSECAfterContact,
	"tr":              periodDateSuffix,
	"tw":              twnicContactFields,
	"uk":              ordinalDates,
	"sn":              domainDNSSECAfterContact,
	"ug":              ugandanDomainUpdate,
	"ve":              dayFirstDates | registrantHandle,
	"xn--4dbrk0ce":    dayFirstDates | changeHistoryDates | domainContactEmail,
	"xn--90a3ac":      dayFirstDates,
	"xn--90ae":        bulgarianStatus,
	"xn--90ais":       domainContactEmail,
	"xn--d1alf":       dayFirstDates | registrantHandle,
	"xn--fiqs8s":      cnnicVariantNames,
	"xn--fiqz9s":      cnnicVariantNames,
	"xn--j6w193g":     dayFirstDates | hkBundledNames,
	"xn--kprw13d":     twnicVariantNames,
	"xn--mgba3a4f16a": registrantHandle,
	"xn--pgbs0dh":     dayFirstDates | domainDNSSECAfterContact,
}

var whoisTimezoneSuffixes = [...]struct{ suffix, offset string }{
	{"(JST)", "+09:00"},
	{"(UTC)", "+00:00"},
	{"(GMT)", "+00:00"},
}

var whoisChileanTimezoneSuffixes = [...]struct{ suffix, offset string }{
	{" CLST", " -03:00"},
	{" CLT", " -04:00"},
}
