package zms

import (
	"golang.org/x/exp/slices"
	"szuro.net/zms/zbx"
)

type Filter struct {
	AcceptedTags []zbx.Tag `yaml:"accepted"`
	RejectedTags []zbx.Tag `yaml:"rejected"`
	active       bool
}

func (f *Filter) Activate() {
	f.active = true
}

// Check if value should be accepted or not
// No tags specified -> everything is acceted
// only AcceptedTags ar eprovided -> only matching tags are allowed
// only RejectedTags are specified -> everything is allowed expect for matching tags
// both AcceptedTags and RejectedTags are provided -> only accepted tags that were not rejected later are accepted
func (f *Filter) EvaluateFilter(tags []zbx.Tag) (accepted bool) {
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
