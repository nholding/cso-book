package audit

import (
	"time"
)

type AuditInfo struct {
	CreatedBy string
	CreatedAt time.Time
	UpdatedBy *string
	UpdatedAt *time.Time
}

// NewAuditInfo returns an AuditInfo with the current timestamp and creator.
func NewAuditInfo(creator string) *AuditInfo {
	if creator == "" {
		creator = "system@internal.local"
	}

	now := time.Now().UTC()

	return &AuditInfo{
		CreatedBy: creator,
		CreatedAt: now,
		UpdatedBy: &creator,
		UpdatedAt: &now,
	}
}

func (a *AuditInfo) UpdateAuditInfo(updatedBy string) {
	if a == nil {
		return // Defensive: nothing to update
	}

	now := time.Now().UTC()

	a.UpdatedBy = &updatedBy
	a.UpdatedAt = &now
}
