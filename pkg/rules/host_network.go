package rules

import "github.com/lowplane/sevro/pkg/parser"

// hostNetwork fires when hostNetwork=true. Host networking flattens
// the pod's network namespace onto the node, exposing localhost-bound
// services on the host network and bypassing NetworkPolicy. Legitimate
// uses are rare (CNI plugins, ingress controllers explicitly designed
// for host networking); most charts that turn it on have done so by
// accident.
type hostNetwork struct{}

func newHostNetwork() Detector { return hostNetwork{} }

func (hostNetwork) ID() string   { return "host-network" }
func (hostNetwork) Name() string { return "hostNetwork enabled" }

func (hostNetwork) Run(w parser.Workload) []Finding {
	if w.Security.HostNetwork == nil || !*w.Security.HostNetwork {
		return nil
	}
	return []Finding{{
		DetectorID: "host-network",
		Workload:   w.Name,
		Title:      "Pod uses host network",
		Detail:     "hostNetwork=true puts the pod in the node's network namespace, bypassing NetworkPolicy and exposing localhost services on the host. Required for some CNI plugins and ingress controllers; otherwise leave it unset.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}
