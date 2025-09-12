package filter

import (
	"golang.org/x/exp/slices"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

type Filter struct {
	AcceptedTags []zbxpkg.Tag `yaml:"accepted"`
	RejectedTags []zbxpkg.Tag `yaml:"rejected"`
	active       bool
}

func (f *Filter) Activate() {
	if len(f.AcceptedTags) != 0 || len(f.RejectedTags) != 0 {
		f.active = true
	}
}

// Check if value should be accepted or not
// No tags specified -> everything is acceted
// only AcceptedTags ar eprovided -> only matching tags are allowed
// only RejectedTags are specified -> everything is allowed expect for matching tags
// both AcceptedTags and RejectedTags are provided -> only accepted tags that were not rejected later are accepted
func (f *Filter) EvaluateFilter(tags []zbxpkg.Tag) (accepted bool) {
	if !f.active {
		return true
	}
	for _, tag := range tags {
		if len(f.AcceptedTags) == 0 {
			accepted = true
			break
		}
		if slices.Contains(f.AcceptedTags, tag) {
			accepted = true
		}
	}

	for _, tag := range tags {
		if slices.Contains(f.RejectedTags, tag) {
			accepted = false
		}
	}
	return
}
