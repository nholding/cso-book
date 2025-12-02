package company

import (
	"strings"

	"github.com/nholding/cso-book/internal/audit"
	"github.com/nholding/cso-book/internal/utils"
)

type Company struct {
	ID              string          `json:"id"`           // Stable ULID (primary key)
	BusinessKey     string          `json:"business_key"` // Deterministic hash for deduplication
	Version         string          `json:"version"`      // ID generation version, e.g. "C1"
	Name            string          `json:"name"`         // Official name, e.g. British Petroleum
	CommonName      string          `json:"common_name"`  // Common name in the market, e.g. BP
	DisplayName     string          `json:"display_name"`
	CoCNumber       string          `json:"coc_number"`
	City            string          `json:"city"`
	Address         string          `json:"address"`
	ContactPersonID string          `json:"contact_person_id"`
	AuditInfo       audit.AuditInfo `json:"audit"`
}

// Generate keys
func (c *Company) GenerateKeys() {
	c.Version = "C1" // version 1 of key logic
	c.ID = utils.GenerateStableID()

	c.BusinessKey = utils.GenerateBusinessKey(c.Version, map[string]string{
		"coc": c.CoCNumber,
	})
}

// CreateCompany creates a company if it doesn't already exist
func NewCompany(name, commonName, displayName, cocNumber, city, address, user string) (Company, error) {
	c := Company{
		Name:        strings.ToLower(name),
		CommonName:  commonName,
		DisplayName: displayName,
		CoCNumber:   cocNumber,
		City:        strings.ToLower(city),
		Address:     strings.ToLower(address),
		AuditInfo:   *audit.NewAuditInfo(user),
	}

	c.GenerateKeys()

	return c, nil
}
