package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lowplane/sevro/internal/rules"
)

func TestText_AlwaysIncludesAccuracyDisclosure(t *testing.T) {
	cases := []Report{
		{Source: "empty", Workloads: 0},
		{Source: "with-findings", Workloads: 1, Findings: []rules.Finding{
			{DetectorID: "x", Workload: "api", Title: "Test finding", Severity: rules.SeverityHigh, Confidence: rules.ConfidenceHigh},
		}},
	}
	for _, r := range cases {
		var buf bytes.Buffer
		if err := Text(&buf, r); err != nil {
			t.Fatalf("Text(%s): %v", r.Source, err)
		}
		if !strings.Contains(buf.String(), AccuracyDisclosure) {
			t.Fatalf("Text(%s): output is missing accuracy disclosure:\n%s", r.Source, buf.String())
		}
	}
}

func TestText_NoFindings(t *testing.T) {
	var buf bytes.Buffer
	r := Report{Source: "demo", Workloads: 5}
	if err := Text(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "No findings across 5 workload(s)") {
		t.Errorf("expected no-findings line, got:\n%s", out)
	}
}

func TestText_RendersFindings(t *testing.T) {
	var buf bytes.Buffer
	r := Report{
		Source:    "fixtures/basic-chart/values.yaml",
		Workloads: 2,
		Findings: []rules.Finding{
			{
				DetectorID:      "cpu-overprovisioned",
				Workload:        "api",
				Title:           "CPU request appears overprovisioned",
				MonthlyUSDCents: 12345,
				Severity:        rules.SeverityMed,
				Confidence:      rules.ConfidenceMed,
			},
			{
				DetectorID: "missing-memory-limit",
				Workload:   "worker",
				Title:      "Memory limit not set",
				Severity:   rules.SeverityHigh,
				Confidence: rules.ConfidenceHigh,
			},
		},
	}
	if err := Text(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"[MED]",
		"CPU request appears overprovisioned",
		"workload: api",
		"[HIGH]",
		"Memory limit not set",
		"$123.45", // monthly_savings (per-finding)
		"Estimated monthly savings: $123.45",
		"sevro.dev",
		"±40%",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestJSON_IncludesDisclosure(t *testing.T) {
	var buf bytes.Buffer
	r := Report{
		Source:    "demo",
		Workloads: 1,
		Findings: []rules.Finding{
			{DetectorID: "cpu-overprovisioned", Workload: "api", Title: "x", MonthlyUSDCents: 100, Severity: rules.SeverityMed, Confidence: rules.ConfidenceMed},
		},
	}
	if err := JSON(&buf, r); err != nil {
		t.Fatal(err)
	}
	var got struct {
		AccuracyDisclosure string  `json:"accuracy_disclosure"`
		MonthlySavingsUSD  float64 `json:"monthly_savings_usd"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, buf.String())
	}
	if got.AccuracyDisclosure != AccuracyDisclosure {
		t.Errorf("disclosure missing or wrong: %q", got.AccuracyDisclosure)
	}
	if got.MonthlySavingsUSD != 1.0 {
		t.Errorf("monthly_savings_usd = %v, want 1.0", got.MonthlySavingsUSD)
	}
}

func TestFormatCents(t *testing.T) {
	cases := map[int64]string{
		0:     "0",
		1:     "0.01",
		100:   "1",
		12345: "123.45",
		99:    "0.99",
	}
	for in, want := range cases {
		if got := formatCents(in); got != want {
			t.Errorf("formatCents(%d) = %q, want %q", in, got, want)
		}
	}
}
