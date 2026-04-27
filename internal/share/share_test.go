package share

import (
	"strings"
	"testing"
)

type sample struct {
	Source    string `json:"source"`
	Workloads int    `json:"workloads_analyzed"`
	Findings  []any  `json:"findings"`
}

func TestHash_Stable(t *testing.T) {
	a := sample{Source: "/tmp/x", Workloads: 3, Findings: []any{
		map[string]any{"DetectorID": "cpu", "Severity": "MED"},
	}}
	h1, err := Hash(a)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := Hash(a)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hash unstable: %q vs %q", h1, h2)
	}
	if len(h1) != HashLen {
		t.Errorf("hash len = %d, want %d", len(h1), HashLen)
	}
}

func TestHash_IgnoresSourcePath(t *testing.T) {
	// Two reports differing only in the source path must hash to the
	// same value — that's the point of sanitisation.
	a := sample{Source: "/home/alice/chart", Workloads: 1}
	b := sample{Source: "/home/bob/chart", Workloads: 1}
	ha, _ := Hash(a)
	hb, _ := Hash(b)
	if ha != hb {
		t.Errorf("source path leaked into hash: %q vs %q", ha, hb)
	}
}

func TestHash_WorkloadsAffectsHash(t *testing.T) {
	a := sample{Workloads: 3}
	b := sample{Workloads: 5}
	ha, _ := Hash(a)
	hb, _ := Hash(b)
	if ha == hb {
		t.Errorf("workload count should change the hash")
	}
}

func TestURL_Format(t *testing.T) {
	u, err := URL(sample{Workloads: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, BaseURL) {
		t.Errorf("URL missing base: %q", u)
	}
	if !IsHash(strings.TrimPrefix(u, BaseURL)) {
		t.Errorf("URL trailer is not a valid hash: %q", u)
	}
}

func TestSanitise_TruncatesLongDetail(t *testing.T) {
	long := strings.Repeat("a", 400)
	in := map[string]any{
		"source":   "/path",
		"findings": []any{map[string]any{"Detail": long}},
	}
	out, err := Sanitise(in)
	if err != nil {
		t.Fatal(err)
	}
	doc := out.(map[string]any)
	if doc["source"] != "(redacted)" {
		t.Errorf("source not redacted: %v", doc["source"])
	}
	finds := doc["findings"].([]any)
	d := finds[0].(map[string]any)["Detail"].(string)
	if len(d) > 260 {
		t.Errorf("detail not truncated: len=%d", len(d))
	}
	if !strings.HasSuffix(d, "…") {
		t.Errorf("truncation marker missing: %q", d)
	}
}

func TestIsHash(t *testing.T) {
	cases := map[string]bool{
		"":             false,
		"abc":          false,
		"abcdef012345": true,
		"ABCDEF012345": true,
		"abcdef01234g": false,
		"abcdef0123456": false, // too long
	}
	for in, want := range cases {
		if got := IsHash(in); got != want {
			t.Errorf("IsHash(%q) = %v, want %v", in, got, want)
		}
	}
}
