package filter

import (
	"slices"

	zbxpkg "zms.szuro.net/pkg/zbx"
)

type GroupFilter struct {
	AcceptedGroups []string `yaml:"accepted"`
	RejectedGroups []string `yaml:"rejected"`
	active         bool
}

func NewGroupFilter(rawFilter FilterConfig) *GroupFilter {
	var f GroupFilter

	f.AcceptedGroups = rawFilter.Accepted
	f.RejectedGroups = rawFilter.Rejected

	if len(f.AcceptedGroups) != 0 || len(f.RejectedGroups) != 0 {
		f.active = true
	}
	return &f
}

func (f *GroupFilter) AcceptHistory(h zbxpkg.History) bool {
	return f.groupFilter(h.Groups)
}
func (f *GroupFilter) AcceptTrend(t zbxpkg.Trend) bool {
	return f.groupFilter(t.Groups)
}
func (f *GroupFilter) AcceptEvent(e zbxpkg.Event) bool {
	return f.groupFilter(e.Groups)
}

func (f *GroupFilter) FilterHistory(h []zbxpkg.History) []zbxpkg.History {
	accepted := make([]zbxpkg.History, 0, len(h))
	for _, H := range h {
		if f.groupFilter(H.Groups) {
			accepted = append(accepted, H)
		}
	}
	return accepted
}
func (f *GroupFilter) FilterTrends(t []zbxpkg.Trend) []zbxpkg.Trend {
	accepted := make([]zbxpkg.Trend, 0, len(t))
	for _, T := range t {
		if f.groupFilter(T.Groups) {
			accepted = append(accepted, T)
		}
	}
	return accepted
}
func (f *GroupFilter) FilterEvents(e []zbxpkg.Event) []zbxpkg.Event {
	accepted := make([]zbxpkg.Event, 0, len(e))
	for _, E := range e {
		if f.groupFilter(E.Groups) {
			accepted = append(accepted, E)
		}
	}
	return accepted
}

func (f *GroupFilter) groupFilter(groups []string) (accepted bool) {
	if !f.active {
		return true
	}

	// Default to false if we have accepted groups (whitelist mode)
	// Default to true if we only have rejected groups (blacklist mode)
	accepted = len(f.AcceptedGroups) == 0

	// If any group is in accepted groups, set accepted = true
	for _, group := range groups {
		if slices.Contains(f.AcceptedGroups, group) {
			accepted = true
			break
		}
	}

	// If any group is in rejected groups, set accepted = false
	for _, group := range groups {
		if slices.Contains(f.RejectedGroups, group) {
			accepted = false
			return
		}
	}

	return
}
