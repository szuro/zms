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

type History struct {
	Host   Host `json:"host"`
	ItemID int  `json:"itemid"`
	Name   string
	Clock  int `json:"clock"`
	Ns     int
	Value  string `json:"value"`
	Type   int
}

type Trend struct {
	Host          Host `json:"host"`
	ItemID        int  `json:"itemid"`
	Name          string
	Clock         int
	Count         int
	Min, Max, Avg float64
	Type          int
}

type Export interface {
	History | Trend
}
