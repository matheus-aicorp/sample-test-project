package domain

import "strings"

// NormalizeSlug lowercases, trims and turns spaces and underscores into
// hyphens. Idempotent, and the single definition of the canonical slug form
// shared by the repository and the HTTP layer.
func NormalizeSlug(raw string) string {
	slug := strings.ToLower(strings.TrimSpace(raw))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}
