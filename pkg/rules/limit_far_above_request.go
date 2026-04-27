package rules

import (
	"fmt"

	"github.com/lowplane/sevro/pkg/parser"
)

// cpuLimitFarAboveRequest and memoryLimitFarAboveRequest fire when the
// limit is many multiples of the request. Kubernetes treats request as
// the scheduling baseline and limit as the cap; a 10× gap means the
// pod can burst hugely beyond what was reserved, which both wrecks
// node capacity planning and lets noisy-neighbour patterns kill
// other workloads on memory pressure.
//
// The threshold is configurable via the constant; production may want
// to tune it once we have measured-utilisation data from agent installs.

const limitBurstRatio = 4 // limit/request > 4 fires

type cpuLimitFarAboveRequest struct{}

func newCPULimitFarAboveRequest() Detector { return cpuLimitFarAboveRequest{} }

func (cpuLimitFarAboveRequest) ID() string   { return "cpu-limit-far-above-request" }
func (cpuLimitFarAboveRequest) Name() string { return "CPU limit far above request" }

func (cpuLimitFarAboveRequest) Run(w parser.Workload) []Finding {
	req, lim := w.Requests.CPU, w.Limits.CPU
	if !req.Set || !lim.Set || req.Value == 0 {
		return nil
	}
	ratio := float64(lim.Value) / float64(req.Value)
	if ratio <= limitBurstRatio {
		return nil
	}
	return []Finding{{
		DetectorID: "cpu-limit-far-above-request",
		Workload:   w.Name,
		Title:      "CPU limit is many multiples of the request",
		Detail:     fmt.Sprintf("Request %s vs limit %s — a %.1fx burst window. The scheduler reserves only the request, so the pod can suddenly consume far more than its node-allocation suggests. Tighten the limit or raise the request to match observed P95.", req, lim, ratio),
		Severity:   SeverityMed,
		Confidence: ConfidenceMed,
	}}
}

type memoryLimitFarAboveRequest struct{}

func newMemoryLimitFarAboveRequest() Detector { return memoryLimitFarAboveRequest{} }

func (memoryLimitFarAboveRequest) ID() string   { return "memory-limit-far-above-request" }
func (memoryLimitFarAboveRequest) Name() string { return "Memory limit far above request" }

func (memoryLimitFarAboveRequest) Run(w parser.Workload) []Finding {
	req, lim := w.Requests.Memory, w.Limits.Memory
	if !req.Set || !lim.Set || req.Value == 0 {
		return nil
	}
	ratio := float64(lim.Value) / float64(req.Value)
	if ratio <= limitBurstRatio {
		return nil
	}
	return []Finding{{
		DetectorID: "memory-limit-far-above-request",
		Workload:   w.Name,
		Title:      "Memory limit is many multiples of the request",
		Detail:     fmt.Sprintf("Request %s vs limit %s — a %.1fx burst window. Memory limits aren't elastic the way CPU is: a pod that climbs to its limit will OOM, taking neighbours down via node memory pressure. Tighten the limit toward observed P95.", req, lim, ratio),
		Severity:   SeverityMed,
		Confidence: ConfidenceMed,
	}}
}
