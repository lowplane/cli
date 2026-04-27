package rules

import "github.com/lowplane/sevro/pkg/parser"

// unboundedImageTag fires when an image is tagged with a known
// floating tag — `main`, `master`, `stable`, `dev`, `staging`,
// `production`, etc. These tags get re-pointed by upstream and
// produce non-reproducible deploys, just like `:latest`.
type unboundedImageTag struct{}

func newUnboundedImageTag() Detector { return unboundedImageTag{} }

func (unboundedImageTag) ID() string   { return "unbounded-image-tag" }
func (unboundedImageTag) Name() string { return "Floating image tag" }

// floatingTags is the closed set of tags we treat as "this points at
// whatever the upstream pushed last". `:latest` is handled by
// image-pinned-latest; this detector covers the rest.
var floatingTags = map[string]bool{
	"main":       true,
	"master":     true,
	"stable":     true,
	"production": true,
	"prod":       true,
	"staging":    true,
	"dev":        true,
	"edge":       true,
	"current":    true,
	"head":       true,
	"trunk":      true,
}

func (unboundedImageTag) Run(w parser.Workload) []Finding {
	if !w.Image.Set {
		return nil
	}
	if !floatingTags[w.Image.Tag] {
		return nil
	}
	return []Finding{{
		DetectorID: "unbounded-image-tag",
		Workload:   w.Name,
		Title:      "Floating image tag prevents reproducible deploys",
		Detail:     "Image tag '" + w.Image.Tag + "' is a floating reference: upstream re-points it whenever they publish. The same Helm release can therefore land different bytes on consecutive deploys, and rollbacks become ambiguous. Pin to an immutable digest or version tag.",
		Severity:   SeverityMed,
		Confidence: ConfidenceHigh,
	}}
}
