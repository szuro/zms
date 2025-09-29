package filter

import (
	"golang.org/x/exp/slices"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

type Filter interface {
	AcceptHistory(h zbxpkg.History) bool
	AcceptTrend(t zbxpkg.Trend) bool
	AcceptEvent(e zbxpkg.Event) bool
	FilterHistory(h []zbxpkg.History) []zbxpkg.History
	FilterTrends(t []zbxpkg.Trend) []zbxpkg.Trend
	FilterEvents(e []zbxpkg.Event) []zbxpkg.Event
}

type DefaultFilter struct {
	AcceptedTags []zbxpkg.Tag `yaml:"accepted"`
	RejectedTags []zbxpkg.Tag `yaml:"rejected"`
	active       bool
}

func NewDefaultFilter(rawFilter map[string]any) *DefaultFilter {
	var f DefaultFilter

	f.RejectedTags = parseTags(rawFilter["rejected"])
	f.AcceptedTags = parseTags(rawFilter["accepted"])

	if len(f.AcceptedTags) != 0 || len(f.RejectedTags) != 0 {
		f.active = true
	}
	return &f
}

func parseTags(rawTags any) []zbxpkg.Tag {
	var tags []zbxpkg.Tag
	if tagSlice, ok := rawTags.([]any); ok {
		for _, item := range tagSlice {
			if tagMap, ok := item.(map[string]any); ok {
				tag := zbxpkg.Tag{}
				if tagName, exists := tagMap["tag"]; exists {
					tag.Tag = tagName.(string)
				}
				if tagValue, exists := tagMap["value"]; exists {
					tag.Value = tagValue.(string)
				}
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func (f *DefaultFilter) AcceptHistory(h zbxpkg.History) bool {
	return f.tagFilter(h.Tags)
}
func (f *DefaultFilter) AcceptTrend(t zbxpkg.Trend) bool {
	return f.tagFilter(t.Tags)
}
func (f *DefaultFilter) AcceptEvent(e zbxpkg.Event) bool {
	return f.tagFilter(e.Tags)
}

func (f *DefaultFilter) FilterHistory(h []zbxpkg.History) []zbxpkg.History {
	accepted := make([]zbxpkg.History, 0, len(h))
	for _, H := range h {
		if f.tagFilter(H.Tags) {
			accepted = append(accepted, H)
		}
	}
	return accepted
}
func (f *DefaultFilter) FilterTrends(t []zbxpkg.Trend) []zbxpkg.Trend {
	accepted := make([]zbxpkg.Trend, 0, len(t))
	for _, T := range t {
		if f.tagFilter(T.Tags) {
			accepted = append(accepted, T)
		}
	}
	return accepted
}
func (f *DefaultFilter) FilterEvents(e []zbxpkg.Event) []zbxpkg.Event {
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
func (f *DefaultFilter) tagFilter(tags []zbxpkg.Tag) (accepted bool) {
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
