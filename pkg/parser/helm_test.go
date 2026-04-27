package parser

import (
	"strings"
	"testing"
)

func TestParseValues_Flat(t *testing.T) {
	in := `
api:
  resources:
    requests:
      cpu: "1"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "4Gi"
worker:
  resources:
    requests:
      cpu: 500m
    limits:
      cpu: 1
`
	wls, err := ParseValues(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseValues: %v", err)
	}
	if len(wls) != 2 {
		t.Fatalf("expected 2 workloads, got %d: %+v", len(wls), wls)
	}

	if wls[0].Name != "api" {
		t.Errorf("wls[0].Name = %q, want api", wls[0].Name)
	}
	if wls[0].Requests.CPU.Value != 1000 {
		t.Errorf("api requests cpu = %d, want 1000", wls[0].Requests.CPU.Value)
	}
	if wls[0].Limits.Memory.Value != 4*1024*1024*1024 {
		t.Errorf("api limits memory = %d, want %d", wls[0].Limits.Memory.Value, 4*1024*1024*1024)
	}

	if wls[1].Name != "worker" {
		t.Errorf("wls[1].Name = %q, want worker", wls[1].Name)
	}
	if wls[1].Requests.Memory.Set {
		t.Errorf("worker requests memory should be unset")
	}
	if wls[1].Limits.Memory.Set {
		t.Errorf("worker limits memory should be unset")
	}
}

func TestParseValues_NestedSubchart(t *testing.T) {
	in := `
postgresql:
  primary:
    resources:
      requests:
        cpu: 2
        memory: 8Gi
`
	wls, err := ParseValues(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseValues: %v", err)
	}
	if len(wls) != 1 {
		t.Fatalf("expected 1 workload, got %+v", wls)
	}
	if wls[0].Name != "postgresql.primary" {
		t.Errorf("nested name = %q, want postgresql.primary", wls[0].Name)
	}
}

func TestParseValues_NoWorkloads(t *testing.T) {
	in := `
config:
  level: info
features:
  - one
  - two
`
	wls, err := ParseValues(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseValues: %v", err)
	}
	if len(wls) != 0 {
		t.Errorf("expected 0 workloads, got %+v", wls)
	}
}

func TestParseValues_BadYAML(t *testing.T) {
	in := "this is: not: valid: yaml::"
	if _, err := ParseValues(strings.NewReader(in)); err == nil {
		t.Fatal("expected error on malformed yaml")
	}
}

func TestParseValues_TopLevelNotMap(t *testing.T) {
	in := "- a\n- b\n"
	if _, err := ParseValues(strings.NewReader(in)); err == nil {
		t.Fatal("expected error when top level is a sequence")
	}
}

func TestParseValues_DeterministicOrder(t *testing.T) {
	in := `
zeta:
  resources:
    requests: {cpu: 1}
alpha:
  resources:
    requests: {cpu: 1}
mike:
  resources:
    requests: {cpu: 1}
`
	wls, err := ParseValues(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "mike", "zeta"}
	for i := range want {
		if wls[i].Name != want[i] {
			t.Errorf("wls[%d].Name = %q, want %q", i, wls[i].Name, want[i])
		}
	}
}
