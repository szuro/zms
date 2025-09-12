package filter

import (
	"testing"

	zbxpkg "szuro.net/zms/pkg/zbx"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		filter    Filter
		tags      []zbxpkg.Tag
		expected  bool
		activated bool
	}{
		{
			name:     "No tags specified, everything accepted",
			filter:   Filter{},
			tags:     []zbxpkg.Tag{},
			expected: true,
		},
		{
			name:     "Only accepted tags provided, matching tag",
			filter:   Filter{AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: true,
		},
		{
			name:     "Only accepted tags provided, non-matching tag",
			filter:   Filter{AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "dev"}},
			expected: false,
		},
		{
			name:     "Only rejected tags provided, non-matching tag",
			filter:   Filter{RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "dev"}},
			expected: true,
		},
		{
			name:     "Only rejected tags provided, matching tag",
			filter:   Filter{RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}}},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: false,
		},
		{
			name: "Both accepted and rejected tags provided, matching accepted tag, non-matching rejected tag",
			filter: Filter{
				AcceptedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
				RejectedTags: []zbxpkg.Tag{{Tag: "role", Value: "test"}},
			},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: true,
		},
		{
			name: "Both accepted and rejected tags provided, matching rejected tag, non-matching accepted tag",
			filter: Filter{
				RejectedTags: []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
				AcceptedTags: []zbxpkg.Tag{{Tag: "role", Value: "test"}},
			},
			tags:     []zbxpkg.Tag{{Tag: "env", Value: "prod"}},
			expected: false,
		},
		{
			name: "Both accepted and rejected tags provided, matching accepted and rejected tags",
			filter: Filter{
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
			result := tt.filter.EvaluateFilter(tt.tags)
			if result != true {
				t.Errorf("EvaluateFilter() = %v, expected %v", result, true)
			}
		})
	}

	t.Log("Testing activated filters")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.Activate()
			result := tt.filter.EvaluateFilter(tt.tags)
			if result != tt.expected {
				t.Errorf("EvaluateFilter() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
