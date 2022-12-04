package zbx

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/nxadm/tail"
)

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
	min, max, avg float64
	Type          int
}

type Export interface {
	History | Trend
}

func parseLine(line *tail.Line) (h History) {
	err := json.Unmarshal([]byte(line.Text), &h)
	if err != nil {
		if h.Type == FLOAT && h.Value == "" {
			h.Value = "0.0"
		} else if h.Type == UNSIGNED && h.Value == "" {
			h.Value = "0"
		}
	}
	return
}

func FileReader(path string, c chan History) {
	t, err := tail.TailFile(
		path, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		fmt.Print(err)
	}

	for line := range t.Lines {
		c <- parseLine(line)
	}
	t.Wait()
}

func FileReaderGenerator(zbx ZabbixConf) (c chan History) {
	c = make(chan History, 100)
	for i := 1; i <= zbx.DBSyncers; i++ {
		filename := filepath.Join(zbx.ExportDir, fmt.Sprintf(HISTORY_EXPORT, i))
		go FileReader(filename, c)
	}
	return
}
