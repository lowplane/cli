package analyze

import (
	"testing"

	"github.com/lowplane/sevro/internal/render"
	"github.com/lowplane/sevro/internal/rules"
)

func sample() render.Report {
	return render.Report{
		Source:    "x",
		Workloads: 3,
		Findings: []rules.Finding{
			{DetectorID: "cpu-overprovisioned", Severity: rules.SeverityMed},
			{DetectorID: "memory-overprovisioned", Severity: rules.SeverityMed},
			{DetectorID: "missing-memory-limit", Severity: rules.SeverityHigh},
			{DetectorID: "missing-cpu-limit", Severity: rules.SeverityLow},
			{DetectorID: "image-pinned-latest", Severity: rules.SeverityMed},
		},
	}
}

func TestFilter_NoOptionsPassthrough(t *testing.T) {
	r := sample()
	out := Filter(r, FilterOptions{})
	if len(out.Findings) != 5 {
		t.Fatalf("len = %d, want 5", len(out.Findings))
	}
}

func TestFilter_SecurityOnly(t *testing.T) {
	out := Filter(sample(), FilterOptions{SecurityOnly: true})
	for _, f := range out.Findings {
		if !SecurityDetectorIDs[f.DetectorID] {
			t.Errorf("non-security detector leaked: %q", f.DetectorID)
		}
	}
	if len(out.Findings) != 3 {
		t.Errorf("len = %d, want 3 (missing-memory-limit + missing-cpu-limit + image-pinned-latest)", len(out.Findings))
	}
}

func TestFilter_MinSeverity(t *testing.T) {
	out := Filter(sample(), FilterOptions{MinSeverity: rules.SeverityMed})
	for _, f := range out.Findings {
		if severityRank(f.Severity) < severityRank(rules.SeverityMed) {
			t.Errorf("LOW finding leaked: %+v", f)
		}
	}
	if len(out.Findings) != 4 {
		t.Errorf("len = %d, want 4", len(out.Findings))
	}
}

func TestFilter_DetectorAllowList(t *testing.T) {
	out := Filter(sample(), FilterOptions{DetectorIDs: []string{"cpu-overprovisioned", "image-pinned-latest"}})
	if len(out.Findings) != 2 {
		t.Fatalf("len = %d, want 2", len(out.Findings))
	}
	got := map[string]bool{}
	for _, f := range out.Findings {
		got[f.DetectorID] = true
	}
	if !got["cpu-overprovisioned"] || !got["image-pinned-latest"] {
		t.Errorf("allow-list leaked or missed: %v", got)
	}
}

func TestFilter_OriginalReportUnchanged(t *testing.T) {
	r := sample()
	before := len(r.Findings)
	_ = Filter(r, FilterOptions{SecurityOnly: true})
	if len(r.Findings) != before {
		t.Errorf("Filter mutated source report: before=%d after=%d", before, len(r.Findings))
	}
}

func TestSeverityRank(t *testing.T) {
	if severityRank(rules.SeverityHigh) <= severityRank(rules.SeverityMed) {
		t.Error("HIGH should outrank MED")
	}
	if severityRank(rules.SeverityMed) <= severityRank(rules.SeverityLow) {
		t.Error("MED should outrank LOW")
	}
	if severityRank(rules.Severity("nonsense")) != 0 {
		t.Error("unknown severity should rank 0")
	}
}
