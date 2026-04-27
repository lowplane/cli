package analyze

import (
	"testing"

	"github.com/lowplane/sevro/pkg/rules"
)

func TestCompute_NoFindings_PerfectScore(t *testing.T) {
	s := Compute("clean", 3, nil)
	if s.Value != 100 {
		t.Errorf("Value = %d, want 100", s.Value)
	}
	if s.Band != rules.ConfidenceHigh {
		t.Errorf("Band = %s, want high", s.Band)
	}
}

func TestCompute_HighSeverityDrops(t *testing.T) {
	s := Compute("dirty", 1, []rules.Finding{
		{DetectorID: "a", Severity: rules.SeverityHigh},
	})
	if s.Value != 100-penaltyHigh {
		t.Errorf("Value = %d, want %d", s.Value, 100-penaltyHigh)
	}
}

func TestCompute_PenaltyCapsAt100(t *testing.T) {
	finds := []rules.Finding{}
	for i := 0; i < 10; i++ {
		finds = append(finds, rules.Finding{DetectorID: "a", Severity: rules.SeverityHigh})
	}
	s := Compute("worst", 1, finds)
	if s.Value != 0 {
		t.Errorf("Value = %d, want 0 (cap)", s.Value)
	}
	if s.Band != rules.ConfidenceLow {
		t.Errorf("Band = %s, want low", s.Band)
	}
}

func TestCompute_BandThresholds(t *testing.T) {
	cases := []struct {
		findings []rules.Finding
		minScore int
		maxScore int
		band     rules.Confidence
	}{
		{nil, 100, 100, rules.ConfidenceHigh},
		{[]rules.Finding{{Severity: rules.SeverityLow}}, 95, 100, rules.ConfidenceHigh},  // 100 - 3 = 97
		{[]rules.Finding{{Severity: rules.SeverityMed}, {Severity: rules.SeverityMed}}, 75, 85, rules.ConfidenceMed}, // 100 - 20 = 80
		{[]rules.Finding{{Severity: rules.SeverityHigh}, {Severity: rules.SeverityHigh}}, 45, 60, rules.ConfidenceLow}, // 100 - 50 = 50
	}
	for i, tc := range cases {
		s := Compute("x", 1, tc.findings)
		if s.Value < tc.minScore || s.Value > tc.maxScore {
			t.Errorf("case %d: Value = %d, want [%d,%d]", i, s.Value, tc.minScore, tc.maxScore)
		}
		if s.Band != tc.band {
			t.Errorf("case %d: Band = %s, want %s", i, s.Band, tc.band)
		}
	}
}

func TestCompute_PenaltiesPerDetector(t *testing.T) {
	s := Compute("x", 1, []rules.Finding{
		{DetectorID: "a", Severity: rules.SeverityHigh},
		{DetectorID: "a", Severity: rules.SeverityMed},
		{DetectorID: "b", Severity: rules.SeverityLow},
	})
	if got, want := s.Penalties["a"], penaltyHigh+penaltyMed; got != want {
		t.Errorf("a penalty = %d, want %d", got, want)
	}
	if got, want := s.Penalties["b"], penaltyLow; got != want {
		t.Errorf("b penalty = %d, want %d", got, want)
	}
}
