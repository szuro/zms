package zbx

import "encoding/json"

const (
	HISTORY_EXPORT  string = "history-history-syncer-%d.ndjson"
	HISTORY_MAIN    string = "history-main-process-0.ndjson"
	TRENDS_EXPORT   string = "trends-history-syncer-%d.ndjson"
	TRENDS_MAIN     string = "trends-main-process-0.ndjson"
	PROBLEMS_EXPORT string = "problems-history-syncer-%d.ndjson"
	PROBLEMS_MAIN   string = "problems-main-process-0.ndjson"
	PROBLEMS_TASK   string = "problems-task-manager-1.ndjson"
)

const (
	FLOAT = iota
	CHARACTER
	LOG
	UNSIGNED
	TEXT
)

const (
	EVENT   = "events"
	HISTORY = "history"
	TREND   = "trends"
)

type Host struct {
	Host string `json:"host"`
	Name string `json:"name"`
}

type Tag struct {
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

type Export interface {
	History | Trend | Event
	ShowTags() []Tag
	GetExportName() string
}

type History struct {
	Host   *Host      `json:"host,omitempty"` // Host name and visible name of the item host
	ItemID int        `json:"itemid"`         // Item ID
	Name   string     // Visible item name
	Clock  int        `json:"clock"` // Number of seconds since Epoch to the moment when value was collected (integer part)
	Groups []string   // List of host groups of the item host
	Ns     int        // Number of nanoseconds to be added to clock to get a precise value collection time
	Value  json.Token `json:"value"`     // Collected item value (number for numeric items, string for text items)
	Tags   []Tag      `json:"item_tags"` // List of item tags (can be empty)
	Type   int        // Collected value type: 0 - numeric float, 1 - character, 2 - log, 3 - numeric unsigned, 4 - text, 5 - binary

	Timestamp int    `json:"timestamp,omitempty"` // Log only. 0 if not available.
	Source    string `json:"source,omitempty"`    // Log only. Source of the log entry. Empty if not available.
	Severity  int    `json:"severity,omitempty"`  // Log only. Severity of the log entry (0 - Not classified, 1 - Information, 2 - Warning, 3 - Average, 4 - High, 5 - Disaster). Empty if not available.
	EventID   int    `json:"eventid,omitempty"`   // Log only. Event ID if available, 0 otherwise.

}

func (h History) ShowTags() []Tag {
	return h.Tags
}

func (h History) GetExportName() string {
	return HISTORY
}

type Trend struct {
	Host          *Host    `json:"host,omitempty"` // Host name and visible name of the item host
	ItemID        int      `json:"itemid"`         // Item ID
	Name          string   // Visible item name
	Clock         int      // Number of seconds since Epoch to the moment when value was collected (integer part)
	Count         int      // Number of values collected for a given hour
	Groups        []string // List of host groups of the item host
	Min, Max, Avg float64  // Minimum, maximum, and average item value for a given hour
	Tags          []Tag    `json:"item_tags"` // List of item tags (can be empty)
	Type          int      // Value type: 0 - numeric float, 3 - numeric unsigned
}

func (t Trend) ShowTags() []Tag {
	return t.Tags
}

func (t Trend) GetExportName() string {
	return TREND
}

type Event struct {
	Clock    int      `json:"clock"`               // Number of seconds since Epoch to the moment when problem was detected or resolved (integer part)
	NS       int      `json:"ns"`                  // Number of nanoseconds to be added to clock to get a precise problem detection/resolution time
	Value    int      `json:"value"`               // 1 for problem, 0 for recovery
	EventID  int      `json:"eventid"`             // Problem or recovery event ID
	PEventID int      `json:"p_eventid,omitempty"` // Problem event ID (for recovery events)
	Name     string   `json:"name,omitempty"`      // Problem event name (problem only)
	Severity int      `json:"severity,omitempty"`  // Problem event severity (problem only): 0 - Not classified, 1 - Information, 2 - Warning, 3 - Average, 4 - High, 5 - Disaster
	Hosts    []Host   `json:"hosts,omitempty"`     // List of hosts involved in the trigger expression (problem only)
	Groups   []string `json:"groups,omitempty"`    // List of host groups of all hosts involved (problem only)
	Tags     []Tag    `json:"tags,omitempty"`      // List of problem tags (problem only, can be empty)
}

func (e Event) ShowTags() []Tag {
	return e.Tags
}

func (e Event) GetExportName() string {
	return EVENT
}
