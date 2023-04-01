package zbx

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

type History struct {
	Host   *Host `json:"host,omitempty"`
	ItemID int   `json:"itemid"`
	Name   string
	Clock  int `json:"clock"`
	Ns     int
	Value  string `json:"value"`
	Type   int
}

type Trend struct {
	Host          *Host `json:"host,omitempty"`
	ItemID        int   `json:"itemid"`
	Name          string
	Clock         int
	Count         int
	Min, Max, Avg float64
	Type          int
}

type Event struct {
	Clock    int      `json:"clock"`
	NS       int      `json:"ns"`
	Value    int      `json:"value"`
	EventID  int      `json:"eventid"`
	PEventID int      `json:"p_eventid"`
	Name     string   `json:"name,omitempty"`
	Severity int      `json:"severity,omitempty"`
	Hosts    []Host   `json:"hosts,omitempty"`
	Groups   []string `json:"groups,omitempty"`
	Tags     []Tag    `json:"tags,omitempty"`
}

type Export interface {
	History | Trend | Event
}
