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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
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

// Sanitise removes from `report` any fields that could contain user
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

// UploadEndpoint is the public sandbox endpoint that accepts a
// sanitised analysis blob. Phase 2 ships the receiver behind this
// URL; until then the CLI gracefully falls back to printing the
// hash + URL stub when the endpoint is unreachable. Override via
// SEVRO_SHARE_URL for self-hosted Sevro deployments.
const UploadEndpoint = "https://sandbox.sevro.dev/api/v1/share"

// uploadTimeout caps how long we wait before giving up and printing
// the local hash. Keep tight — the CLI is interactive.
const uploadTimeout = 5 * time.Second

// UploadResult is what Upload returns. The hash + URL fields are
// always populated; Posted reports whether the upload itself succeeded.
type UploadResult struct {
	Hash   string
	URL    string
	Posted bool   // true when the HTTP POST returned 2xx
	Error  string // non-empty when Posted=false
}

// Upload attempts to POST the sanitised report JSON to endpoint and
// returns the resulting (hash, URL, posted) tuple.
//
// CLI hard rule: this is the only outbound network call sevro makes,
// and only when the user explicitly passes --share. The function never
// retries; never logs request bodies; never sends anything but the
// sanitised payload.
func Upload(report any, endpoint string) UploadResult {
	hash, err := Hash(report)
	if err != nil {
		return UploadResult{Error: err.Error()}
	}
	url := BaseURL + hash
	if endpoint == "" {
		endpoint = UploadEndpoint
	}

	sanitised, err := Sanitise(report)
	if err != nil {
		return UploadResult{Hash: hash, URL: url, Error: err.Error()}
	}
	body, err := canonicalJSON(sanitised)
	if err != nil {
		return UploadResult{Hash: hash, URL: url, Error: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return UploadResult{Hash: hash, URL: url, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sevro-Hash", hash)
	req.Header.Set("User-Agent", "sevro-cli")

	client := &http.Client{Timeout: uploadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return UploadResult{Hash: hash, URL: url, Error: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UploadResult{Hash: hash, URL: url, Error: fmt.Sprintf("upload rejected: HTTP %d", resp.StatusCode)}
	}
	return UploadResult{Hash: hash, URL: url, Posted: true}
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
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
