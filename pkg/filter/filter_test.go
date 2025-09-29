package filter

import (
	"testing"

	zbxpkg "szuro.net/zms/pkg/zbx"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		filter    DefaultFilter
		tags      []zbxpkg.Tag
		expected  bool
		activated bool
	}{
		{
			name:     "No tags specified, everything accepted",
			filter:   DefaultFilter{},
			tags:     []zbxpkg.Tag{},
			expected: true,
		},
		{
			name:     "Only accepted tags provided, matching tag",
			filter:   DefaultFilter{AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: true,
		},
		{
			name:     "Only accepted tags provided, non-matching tag",
			filter:   DefaultFilter{AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "dev"}},
			expected: false,
		},
		{
			name:     "Only rejected tags provided, non-matching tag",
			filter:   DefaultFilter{RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "dev"}},
			expected: true,
		},
		{
			name:     "Only rejected tags provided, matching tag",
			filter:   DefaultFilter{RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: false,
		},
		{
			name: "Both accepted and rejected tags provided, matching accepted tag, non-matching rejected tag",
			filter: DefaultFilter{
				AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
				RejectedTags: []zbxpkg.Tag{{Tag: "role", Value: "test"}},
			},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: true,
		},
		{
			name: "Both accepted and rejected tags provided, matching rejected tag, non-matching accepted tag",
			filter: DefaultFilter{
				RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
				AcceptedTags: []zbxpkg.Tag{{Tag: "role", Value: "test"}},
			},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: false,
		},
		{
			name: "Both accepted and rejected tags provided, matching accepted and rejected tags",
			filter: DefaultFilter{
				AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
				RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: false,
		},
	}

	t.Log("Testing unactivated filters")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy history record with the test tags
			h := zbxpkg.History{Tags: tt.tags}
			result := tt.filter.AcceptHistory(h)
			if result != true {
				t.Errorf("AcceptHistory() = %v, expected %v", result, true)
			}
		})
	}

	t.Log("Testing activated filters")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Activate filter by setting the active flag manually since we have tags
			if len(tt.filter.AcceptedTags) != 0 || len(tt.filter.RejectedTags) != 0 {
				// Use reflection to set the private active field or create a new filter
				activatedFilter := DefaultFilter{
					AcceptedTags: tt.filter.AcceptedTags,
					RejectedTags: tt.filter.RejectedTags,
				}
				// Manually set active flag since we know we have tags
				activatedFilter.active = true

				// Create a dummy history record with the test tags
				h := zbxpkg.History{Tags: tt.tags}
				result := activatedFilter.AcceptHistory(h)
				if result != tt.expected {
					t.Errorf("AcceptHistory() = %v, expected %v", result, tt.expected)
				}
			} else {
				// No tags means filter should be inactive
				h := zbxpkg.History{Tags: tt.tags}
				result := tt.filter.AcceptHistory(h)
				if result != true {
					t.Errorf("AcceptHistory() = %v, expected %v", result, true)
				}
			}
		})
	}
}

func TestFilterTrends(t *testing.T) {
	filter := DefaultFilter{
		AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
		RejectedTags: []zbxpkg.Tag{{Tag: "role", Value: "test"}},
		active:       true, // manually activate since we have tags
	}

	// Test trend acceptance
	trend := zbxpkg.Trend{Tags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}}
	if !filter.AcceptTrend(trend) {
		t.Error("Expected trend to be accepted")
	}

	// Test trend rejection
	trend = zbxpkg.Trend{Tags: []zbxpkg.Tag{{Tag: "role", Value: "test"}}}
	if filter.AcceptTrend(trend) {
		t.Error("Expected trend to be rejected")
	}
}

func TestFilterEvents(t *testing.T) {
	filter := DefaultFilter{
		AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
		RejectedTags: []zbxpkg.Tag{{Tag: "role", Value: "test"}},
		active:       true, // manually activate since we have tags
	}

	// Test event acceptance
	event := zbxpkg.Event{Tags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}}
	if !filter.AcceptEvent(event) {
		t.Error("Expected event to be accepted")
	}

	// Test event rejection
	event = zbxpkg.Event{Tags: []zbxpkg.Tag{{Tag: "role", Value: "test"}}}
	if filter.AcceptEvent(event) {
		t.Error("Expected event to be rejected")
	}
}

func TestFilterSlices(t *testing.T) {
	filter := DefaultFilter{
		AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
		active:       true, // manually activate since we have tags
	}

	// Test FilterHistory
	histories := []zbxpkg.History{
		{Tags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
		{Tags: []zbxpkg.Tag{{Tag: "env", Value: "dev"}}},
	}
	filtered := filter.FilterHistory(histories)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 history record, got %d", len(filtered))
	}

	// Test FilterTrends
	trends := []zbxpkg.Trend{
		{Tags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
		{Tags: []zbxpkg.Tag{{Tag: "env", Value: "dev"}}},
	}
	filteredTrends := filter.FilterTrends(trends)
	if len(filteredTrends) != 1 {
		t.Errorf("Expected 1 trend record, got %d", len(filteredTrends))
	}

	// Test FilterEvents
	events := []zbxpkg.Event{
		{Tags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
		{Tags: []zbxpkg.Tag{{Tag: "env", Value: "dev"}}},
	}
	filteredEvents := filter.FilterEvents(events)
	if len(filteredEvents) != 1 {
		t.Errorf("Expected 1 event record, got %d", len(filteredEvents))
	}
}
