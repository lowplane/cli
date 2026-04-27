package rules

import (
	"fmt"

	"github.com/lowplane/sevro/pkg/parser"
)

// cpuRequestEqualsLimit and memoryRequestEqualsLimit fire when
// request == limit, putting the pod in `Guaranteed` QoS class.
//
// Guaranteed QoS is the highest-eviction-priority class — useful for
// SLO-bound services that must not be killed under pressure — but it
// also means the pod has zero burst headroom. Charts that set both
// values equal "for safety" without intent waste burst capacity.
//
// We surface as INFO/LOW: it's a posture choice, not a bug.

type cpuRequestEqualsLimit struct{}

func newCPURequestEqualsLimit() Detector { return cpuRequestEqualsLimit{} }

func (cpuRequestEqualsLimit) ID() string   { return "cpu-request-equals-limit" }
func (cpuRequestEqualsLimit) Name() string { return "CPU request equals limit (Guaranteed QoS)" }

func (cpuRequestEqualsLimit) Run(w parser.Workload) []Finding {
	if !w.Requests.CPU.Set || !w.Limits.CPU.Set {
		return nil
	}
	if w.Requests.CPU.Value != w.Limits.CPU.Value {
		return nil
	}
	return []Finding{{
		DetectorID: "cpu-request-equals-limit",
		Workload:   w.Name,
		Title:      "CPU request equals limit (Guaranteed QoS)",
		Detail:     fmt.Sprintf("Request and limit both set to %s, putting the pod in Guaranteed QoS — top eviction priority but zero burst headroom. Confirm this is intentional for an SLO-bound workload; otherwise lower the request to allow the scheduler to bin-pack more densely.", w.Requests.CPU),
		Severity:   SeverityLow,
		Confidence: ConfidenceMed,
	}}
}

type memoryRequestEqualsLimit struct{}

func newMemoryRequestEqualsLimit() Detector { return memoryRequestEqualsLimit{} }

func (memoryRequestEqualsLimit) ID() string   { return "memory-request-equals-limit" }
func (memoryRequestEqualsLimit) Name() string { return "Memory request equals limit (Guaranteed QoS)" }

func (memoryRequestEqualsLimit) Run(w parser.Workload) []Finding {
	if !w.Requests.Memory.Set || !w.Limits.Memory.Set {
		return nil
	}
	if w.Requests.Memory.Value != w.Limits.Memory.Value {
		return nil
	}
	return []Finding{{
		DetectorID: "memory-request-equals-limit",
		Workload:   w.Name,
		Title:      "Memory request equals limit (Guaranteed QoS)",
		Detail:     fmt.Sprintf("Memory request and limit both set to %s. Memory is non-burstable anyway, so this only affects QoS class; for SLO-bound workloads it's correct, otherwise lower the request to ~P95 to free node capacity for other pods.", w.Requests.Memory),
		Severity:   SeverityLow,
		Confidence: ConfidenceMed,
	}}
}
