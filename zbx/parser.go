package zbx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/nxadm/tail"
)

func parseHistoryLine(line *tail.Line) (h History, err error) {
	err = json.Unmarshal([]byte(line.Text), &h)
	if err != nil {
		if h.Type == FLOAT && h.Value == "" {
			h.Value = "0.0"
		} else if h.Type == UNSIGNED && h.Value == "" {
			h.Value = "0"
		}
	}
	return
}

func parseTrendLine(line *tail.Line) (t Trend, err error) {
	err = json.Unmarshal([]byte(line.Text), &t)
	return
}

func parseLine[T Export](line *tail.Line) (any, error) {
	var t T
	switch any(t).(type) {
	case History:
		return parseHistoryLine(line)
	case Trend:
		return parseTrendLine(line)
	}
	return nil, errors.New("not a supported export type")
}

func getBasePath[T Export]() (p string) {
	var t T
	switch any(t).(type) {
	case History:
		p = HISTORY_EXPORT
	case Trend:
		p = TRENDS_EXPORT
	}

	return
}

func FileReaderGenerator[T Export](zbx ZabbixConf) (c chan any) {
	c = make(chan any, 100)
	for i := 1; i <= zbx.DBSyncers; i++ {
		filename := filepath.Join(zbx.ExportDir, fmt.Sprintf(getBasePath[T](), i))
		go func() {
			log.Printf("Opening %s...\n", filename)
			t, err := tail.TailFile(
				filename, tail.Config{Follow: true, ReOpen: true})
			if err != nil {
				log.Printf("Fail! Could not open %s. Error: %s\n", filename, err)
				return
			}
			log.Printf("Success! %s opened. Parsing...\n", filename)
			for line := range t.Lines {
				parsed, err := parseLine[T](line)
				if err != nil {
					continue
				}
				c <- parsed
			}
			t.Wait()
		}()
	}

	return
}
