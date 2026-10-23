package output

import (
	"strings"

	"github.com/sd800/domain-check/internal/registry"
)

func registrantEntries(c *registry.RegistrantContact) []entry {
	registrant, organization := c.DisplayNames()
	rows := []entry{{6, registrant}, {23, organization}, {12, strings.Join(c.Address, "\n")}, {24, strings.Join(c.Street, "\n")}, {19, c.City}, {25, c.StateProvince}, {21, c.PostalCode}, {26, c.Country}}
	for _, phone := range c.Phones {
		rows = append(rows, entry{27, phone})
	}
	for _, email := range c.Emails {
		rows = append(rows, entry{28, email})
	}
	return rows
}
