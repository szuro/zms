package filter

import zbxpkg "zms.szuro.net/pkg/zbx"

type EmptyFilter struct{}

func NewEmptytFilter() *EmptyFilter {
	var f EmptyFilter
	return &f
}

func (f *EmptyFilter) AcceptHistory(h zbxpkg.History) bool {
	return true
}
func (f *EmptyFilter) AcceptTrend(t zbxpkg.Trend) bool {
	return true
}
func (f *EmptyFilter) AcceptEvent(e zbxpkg.Event) bool {
	return true
}

func (f *EmptyFilter) FilterHistory(h []zbxpkg.History) []zbxpkg.History {
	return h
}

func (f *EmptyFilter) FilterTrends(t []zbxpkg.Trend) []zbxpkg.Trend {
	return t
}

func (f *EmptyFilter) FilterEvents(e []zbxpkg.Event) []zbxpkg.Event {
	return e
}
