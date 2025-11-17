package filter

import (
	"strings"

	"golang.org/x/exp/slices"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

type TagFilter struct {
	AcceptedTags []zbxpkg.Tag `yaml:"accepted"`
	RejectedTags []zbxpkg.Tag `yaml:"rejected"`
	active       bool
}

func NewTagFilter(rawFilter FilterConfig) *TagFilter {
	var f TagFilter

	f.RejectedTags = parseTags(rawFilter.Rejected)
	f.AcceptedTags = parseTags(rawFilter.Accepted)

	if len(f.AcceptedTags) != 0 || len(f.RejectedTags) != 0 {
		f.active = true
	}
	return &f
}

func parseTags(rawTags []string) []zbxpkg.Tag {
	var tags []zbxpkg.Tag
	for _, tag := range rawTags {
		splitTag := strings.Split(tag, ":")
		tags = append(tags, zbxpkg.Tag{
			Tag:   splitTag[0],
			Value: splitTag[1],
		})
	}
	return tags
}

func (f *TagFilter) AcceptHistory(h zbxpkg.History) bool {
	return f.tagFilter(h.Tags)
}
func (f *TagFilter) AcceptTrend(t zbxpkg.Trend) bool {
	return f.tagFilter(t.Tags)
}
func (f *TagFilter) AcceptEvent(e zbxpkg.Event) bool {
	return f.tagFilter(e.Tags)
}

func (f *TagFilter) FilterHistory(h []zbxpkg.History) []zbxpkg.History {
	accepted := make([]zbxpkg.History, 0, len(h))
	for _, H := range h {
		if f.tagFilter(H.Tags) {
			accepted = append(accepted, H)
		}
	}
	return accepted
}
func (f *TagFilter) FilterTrends(t []zbxpkg.Trend) []zbxpkg.Trend {
	accepted := make([]zbxpkg.Trend, 0, len(t))
	for _, T := range t {
		if f.tagFilter(T.Tags) {
			accepted = append(accepted, T)
		}
	}
	return accepted
}
func (f *TagFilter) FilterEvents(e []zbxpkg.Event) []zbxpkg.Event {
	accepted := make([]zbxpkg.Event, 0, len(e))
	for _, E := range e {
		if f.tagFilter(E.Tags) {
			accepted = append(accepted, E)
		}
	}
	return accepted
}

// Check if value should be accepted or not
// No tags specified -> everything is acceted
// only AcceptedTags ar eprovided -> only matching tags are allowed
// only RejectedTags are specified -> everything is allowed expect for matching tags
// both AcceptedTags and RejectedTags are provided -> only accepted tags that were not rejected later are accepted
func (f *TagFilter) tagFilter(tags []zbxpkg.Tag) (accepted bool) {
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
