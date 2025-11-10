package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"sort"
	"strings"

	"github.com/oklog/ulid/v2"
)

func GenerateStableID() string {
	return ulid.Make().String()
}

// GenerateBusinessKey creates a deterministic, versioned hash for deduplication.
func GenerateBusinessKey(version string, fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for _, k := range keys {
		canonical.WriteString(strings.ToLower(strings.TrimSpace(fields[k])) + "|")
	}

	hash := sha256.Sum256([]byte(canonical.String()))
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])

	return version + "_" + encoded
}
