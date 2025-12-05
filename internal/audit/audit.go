package audit

import (
	"time"
)

type AuditInfo struct {
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by, omitempty"`
	UpdatedAt time.Time `json:"updated_at, omitempty"`
}

// NewAuditInfo returns an AuditInfo with the current timestamp and creator.
func NewAuditInfo(creator string) *AuditInfo {
	var c string
	if creator != "" {
		c = creator
	} else {
		c = "system"
	}

	return &AuditInfo{
		CreatedBy: c,
		CreatedAt: time.Now().UTC(),
	}
}

func (a *AuditInfo) UpdateAuditInfo(updatedBy string) {
	a.UpdatedBy = updatedBy
	a.UpdatedAt = time.Now().UTC()
}
