// Package zbx provides types and interfaces for handling Zabbix export data.
//
// This package defines the core data structures for processing Zabbix exports
// including History, Trends, and Events. It provides a common interface (Export)
// that allows for generic processing of different export types.
//
// The package supports three main export types:
//   - History: Individual metric values collected from monitored items
//   - Trends: Aggregated hourly statistics (min, max, avg) for numeric items
//   - Events: Problem and recovery events from Zabbix triggers
//
// Example usage:
//
//	import "zms.szuro.net/pkg/zbx"
//
//	// Process any export type generically
//	func processExport[T zbx.Export](exports []T) {
//	    for _, export := range exports {
//	        tags := export.ShowTags()
//	        exportType := export.GetExportName()
//	        hash := export.Hash()
//	        // Process the export...
//	    }
//	}
//
//	// Check if history value is numeric
//	if history.IsNumeric() {
//	    // Process numeric value
//	}
package zbx

import (
	"encoding/json"
	"fmt"
)

// File naming constants for Zabbix export files.
// These patterns are used to identify and locate export files created by Zabbix server.
const (
	// HISTORY_EXPORT is the filename pattern for history data exported by DB syncers.
	// The %d placeholder is replaced with the syncer number.
	HISTORY_EXPORT string = "history-history-syncer-%d.ndjson"

	// HISTORY_MAIN is the filename for history data from the main Zabbix process.
	HISTORY_MAIN string = "history-main-process-0.ndjson"

	// TRENDS_EXPORT is the filename pattern for trend data exported by DB syncers.
	// The %d placeholder is replaced with the syncer number.
	TRENDS_EXPORT string = "trends-history-syncer-%d.ndjson"

	// TRENDS_MAIN is the filename for trend data from the main Zabbix process.
	TRENDS_MAIN string = "trends-main-process-0.ndjson"

	// PROBLEMS_EXPORT is the filename pattern for problem events exported by DB syncers.
	// The %d placeholder is replaced with the syncer number.
	PROBLEMS_EXPORT string = "problems-history-syncer-%d.ndjson"

	// PROBLEMS_MAIN is the filename for problem events from the main Zabbix process.
	PROBLEMS_MAIN string = "problems-main-process-0.ndjson"

	// PROBLEMS_TASK is the filename for problem events from the task manager process.
	PROBLEMS_TASK string = "problems-task-manager-1.ndjson"
)

// Value type constants for Zabbix item values.
// These correspond to the different data types that Zabbix can collect and store.
const (
	// FLOAT represents numeric floating-point values (Zabbix value type 0).
	FLOAT = iota

	// CHARACTER represents character/string values (Zabbix value type 1).
	CHARACTER

	// LOG represents log file entries (Zabbix value type 2).
	LOG

	// UNSIGNED represents numeric unsigned integer values (Zabbix value type 3).
	UNSIGNED

	// TEXT represents text values (Zabbix value type 4).
	TEXT
)

// Export type constants that identify the different types of Zabbix exports.
// These are used by the Export interface to identify the export type.
const (
	// EVENT identifies event/problem exports.
	EVENT = "events"

	// HISTORY identifies history data exports.
	HISTORY = "history"

	// TREND identifies trend data exports.
	TREND = "trends"
)

const (
	TREND_AVG   = "avg"
	TREND_MIN   = "min"
	TREND_MAX   = "max"
	TREND_COUNT = "count"
)

// Host represents a Zabbix host with its technical name and display name.
type Host struct {
	// Host is the technical host name used internally by Zabbix.
	Host string `json:"host"`

	// Name is the visible/display name of the host as shown in Zabbix frontend.
	Name string `json:"name"`
}

// Tag represents a key-value pair tag associated with Zabbix items or problems.
// Tags are used for categorization, filtering, and metadata purposes.
type Tag struct {
	// Tag is the tag name/key.
	Tag string `json:"tag"`

	// Value is the tag value.
	Value string `json:"value"`
}

// Export is a generic interface that all Zabbix export types implement.
// It provides a common way to handle different export types (History, Trend, Event)
// in a type-safe manner using Go generics.
//
// The interface includes a type constraint that limits implementations to the
// three supported export types and defines methods for accessing tags,
// identifying the export type, and generating unique hashes.
type Export interface {
	// Type constraint limiting implementations to supported export types.
	History | Trend | Event

	// ShowTags returns the tags associated with this export.
	// Tags are used for filtering and categorization.
	ShowTags() []Tag

	// GetExportName returns the string identifier for this export type.
	// Returns one of: "history", "trends", or "events".
	GetExportName() string

	// Hash generates a unique byte slice identifier for this export record.
	// Used for deduplication and buffering operations.
	Hash() []byte
}

// History represents a single Zabbix history record containing a collected item value.
// History records are created when Zabbix collects data from monitored items and
// contain the actual metric values along with metadata about when and where they were collected.
type History struct {
	// Host contains the technical and display names of the host that owns this item.
	// May be nil in some export configurations.
	Host *Host `json:"host,omitempty"`

	// ItemID is the unique identifier of the Zabbix item.
	ItemID int `json:"itemid"`

	// Name is the visible/display name of the item as shown in Zabbix frontend.
	Name string

	// Clock is the Unix timestamp (seconds since epoch) when the value was collected.
	// This represents the integer part of the collection time.
	Clock int `json:"clock"`

	// Groups contains the list of host groups that the item's host belongs to.
	Groups []string

	// Ns is the nanoseconds component to be added to Clock for precise timing.
	// Together with Clock, this provides nanosecond-precision timestamps.
	Ns int

	// Value contains the actual collected item value. The type depends on the item:
	// - Numeric items: json.Number
	// - Text items: string
	// Use Type field to determine how to interpret this value.
	Value json.Token `json:"value"`

	// Tags contains the list of tags associated with this item.
	// Tags can be used for filtering and categorization. May be empty.
	Tags []Tag `json:"item_tags"`

	// Type indicates the value type using Zabbix value type constants:
	// 0 (FLOAT) - numeric floating-point value
	// 1 (CHARACTER) - character/string value
	// 2 (LOG) - log file entry
	// 3 (UNSIGNED) - numeric unsigned integer value
	// 4 (TEXT) - text value
	Type int

	// Log-specific fields (only present when Type == LOG):

	// Timestamp is the original timestamp from the log entry (log items only).
	// Set to 0 if not available.
	Timestamp int `json:"timestamp,omitempty"`

	// Source is the source of the log entry (log items only).
	// Empty string if not available.
	Source string `json:"source,omitempty"`

	// Severity represents the log entry severity level (log items only):
	// 0 - Not classified
	// 1 - Information
	// 2 - Warning
	// 3 - Average
	// 4 - High
	// 5 - Disaster
	Severity int `json:"severity,omitempty"`

	// EventID is the related event ID for log entries (log items only).
	// Set to 0 if not available.
	EventID int `json:"eventid,omitempty"`
}

// ShowTags returns the tags associated with this history record.
// Implements the Export interface.
func (h History) ShowTags() []Tag {
	return h.Tags
}

// GetExportName returns "history" to identify this as a history export.
// Implements the Export interface.
func (h History) GetExportName() string {
	return HISTORY
}

// Hash generates a unique identifier for this history record.
// The hash is based on ItemID, Clock, and Ns to ensure uniqueness.
// Implements the Export interface.
func (h History) Hash() []byte {
	return []byte("history_" + fmt.Sprint(h.ItemID) + ":" + fmt.Sprint(h.Clock) + ":" + fmt.Sprint(h.Ns))
}

// IsNumeric returns true if this history record contains a numeric value.
// Returns true for FLOAT (0) and UNSIGNED (3) value types.
// Numeric values can be processed mathematically and used in calculations.
func (h History) IsNumeric() bool {
	return h.Type == FLOAT || h.Type == UNSIGNED
}

// Trend represents aggregated hourly statistics for a Zabbix item.
// Trends are generated by Zabbix for numeric items to provide statistical summaries
// over time periods, reducing storage requirements while maintaining useful metrics.
type Trend struct {
	// Host contains the technical and display names of the host that owns this item.
	// May be nil in some export configurations.
	Host *Host `json:"host,omitempty"`

	// ItemID is the unique identifier of the Zabbix item.
	ItemID int `json:"itemid"`

	// Name is the visible/display name of the item as shown in Zabbix frontend.
	Name string

	// Clock is the Unix timestamp (seconds since epoch) representing the hour
	// for which these trend statistics were calculated.
	Clock int

	// Count is the number of individual values that were collected and
	// aggregated to calculate these trend statistics.
	Count int

	// Groups contains the list of host groups that the item's host belongs to.
	Groups []string

	// Min is the minimum value collected during this hour.
	Min float64

	// Max is the maximum value collected during this hour.
	Max float64

	// Avg is the average value calculated from all values collected during this hour.
	Avg float64

	// Tags contains the list of tags associated with this item.
	// Tags can be used for filtering and categorization. May be empty.
	Tags []Tag `json:"item_tags"`

	// Type indicates the original value type of the item:
	// 0 (FLOAT) - numeric floating-point values
	// 3 (UNSIGNED) - numeric unsigned integer values
	// Note: Only numeric types have trend data generated.
	Type int
}

// ShowTags returns the tags associated with this trend record.
// Implements the Export interface.
func (t Trend) ShowTags() []Tag {
	return t.Tags
}

// GetExportName returns "trends" to identify this as a trend export.
// Implements the Export interface.
func (t Trend) GetExportName() string {
	return TREND
}

// Hash generates a unique identifier for this trend record.
// The hash is based on ItemID and Clock (hour) to ensure uniqueness.
// Implements the Export interface.
func (t Trend) Hash() []byte {
	return []byte("trend_" + fmt.Sprint(t.ItemID) + ":" + fmt.Sprint(t.Clock))
}

// Event represents a Zabbix problem or recovery event.
// Events are generated when triggers change state, either from OK to PROBLEM (problem events)
// or from PROBLEM to OK (recovery events). They contain information about what happened,
// when it occurred, and which hosts and groups were affected.
type Event struct {
	// Clock is the Unix timestamp (seconds since epoch) when the problem was
	// detected or resolved. This represents the integer part of the event time.
	Clock int `json:"clock"`

	// NS is the nanoseconds component to be added to Clock for precise timing.
	// Together with Clock, this provides nanosecond-precision timestamps.
	NS int `json:"ns"`

	// Value indicates the event type:
	// 1 - problem event (trigger went from OK to PROBLEM)
	// 0 - recovery event (trigger went from PROBLEM to OK)
	Value int `json:"value"`

	// EventID is the unique identifier for this specific event.
	EventID int `json:"eventid"`

	// PEventID is the ID of the related problem event (for recovery events only).
	// This links recovery events back to their corresponding problem events.
	PEventID int `json:"p_eventid,omitempty"`

	// Name is the descriptive name of the problem (problem events only).
	// This is typically the trigger expression or a user-defined description.
	Name string `json:"name,omitempty"`

	// Severity indicates the severity level of the problem (problem events only):
	// 0 - Not classified
	// 1 - Information
	// 2 - Warning
	// 3 - Average
	// 4 - High
	// 5 - Disaster
	Severity int `json:"severity,omitempty"`

	// Hosts contains the list of hosts involved in the trigger expression that
	// generated this event (problem events only).
	Hosts []Host `json:"hosts,omitempty"`

	// Groups contains the list of host groups for all hosts involved in this
	// event (problem events only).
	Groups []string `json:"groups,omitempty"`

	// Tags contains the list of problem tags associated with this event
	// (problem events only). Tags can be used for filtering and categorization.
	Tags []Tag `json:"tags,omitempty"`
}

// ShowTags returns the tags associated with this event record.
// Implements the Export interface.
func (e Event) ShowTags() []Tag {
	return e.Tags
}

// GetExportName returns "events" to identify this as an event export.
// Implements the Export interface.
func (e Event) GetExportName() string {
	return EVENT
}

// Hash generates a unique identifier for this event record.
// The hash is based on EventID to ensure uniqueness.
// Implements the Export interface.
func (e Event) Hash() []byte {
	return []byte("event_" + fmt.Sprint(e.EventID))
}
