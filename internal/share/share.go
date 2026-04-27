// Package share computes a content-addressable hash of an analysis
// payload and produces a `sevro.dev/r/<hash>` URL.
//
// The CLI's `--share` flag is the only network egress sevro takes
// in Phase 1. Even then, Phase 1 ships only the local hashing — the
// actual upload endpoint lives behind the public sandbox (Phase 2).
// Until that lands, the CLI prints the hash + URL so users get a
// stable identifier they can manually copy into a sevro.dev/r/<hash>
// page once the endpoint is live.
//
// Hard rules (from CLAUDE.md):
//   - No telemetry by default. The CLI must never call out unless
//     the user explicitly passes --share.
//   - PII minimisation. We hash a *sanitised* payload — file paths
//     and user identifiers are stripped before hashing.
package share

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// BaseURL is the public share host. Hashes resolve to a read-only HTML
// view of the analysis once Phase 2 ships the sandbox; until then the
// URL is informational.
const BaseURL = "https://sevro.dev/r/"

// HashLen controls how many hex characters of the SHA-256 digest the
// CLI exposes in the URL. 12 chars (48 bits) is collision-safe at the
// query volumes Phase 1 expects; larger digests stay available via the
// JSON output.
const HashLen = 12

// Sanitised removes from `report` any fields that could contain user
// or environment-specific data before hashing. The output has the
// same shape as the original report; consumers should not assume
// payload bytes are equal between runs that differ only in source
// path.
//
// Specifically we strip:
//   - report.source — replaced with "(redacted)"
//   - finding.detail content past the first 256 chars (caps payload
//     size; details are deterministic per finding so this is safe)
//
// Anything else is kept; the values themselves are the analysis
// signature we're hashing.
func Sanitise(report any) (any, error) {
	raw, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("share: marshal: %w", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("share: unmarshal: %w", err)
	}
	if _, ok := doc["source"]; ok {
		doc["source"] = "(redacted)"
	}
	if findings, ok := doc["findings"].([]any); ok {
		for _, f := range findings {
			fm, ok := f.(map[string]any)
			if !ok {
				continue
			}
			if d, ok := fm["Detail"].(string); ok && len(d) > 256 {
				fm["Detail"] = d[:256] + "…"
			}
		}
	}
	return doc, nil
}

// Hash returns the hex-encoded SHA-256 of the sanitised JSON encoding
// of report, truncated to HashLen characters. Stable: same input always
// produces the same hash, regardless of map ordering (we use Marshal's
// sorted-key emission).
func Hash(report any) (string, error) {
	sanitised, err := Sanitise(report)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalJSON(sanitised)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])[:HashLen], nil
}

// URL returns the full sevro.dev/r/<hash> URL for the report.
func URL(report any) (string, error) {
	h, err := Hash(report)
	if err != nil {
		return "", err
	}
	return BaseURL + h, nil
}

// canonicalJSON marshals v with deterministic key ordering. Go's
// encoding/json already sorts map keys, but nested types may not — we
// re-marshal through map[string]any to ensure stability.
func canonicalJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}

// IsHash reports whether s looks like a hash this package would emit.
// Useful for accepting `sevro <subcommand> --share-hash <id>` reads
// in future phases.
func IsHash(s string) bool {
	if len(s) != HashLen {
		return false
	}
	s = strings.ToLower(s)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
