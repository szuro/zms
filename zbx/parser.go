package zbx

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"log/slog"

	"github.com/nxadm/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

func parseEventLine(line *tail.Line) (e Event, err error) {
	err = json.Unmarshal([]byte(line.Text), &e)
	return
}

func parseLine[T Export](line *tail.Line) (any, error) {
	var t T
	switch any(t).(type) {
	case History:
		return parseHistoryLine(line)
	case Trend:
		return parseTrendLine(line)
	case Event:
		return parseEventLine(line)
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
	case Event:
		p = PROBLEMS_EXPORT
	}

	return
}

func FileReaderGenerator[T Export](zbx ZabbixConf) (c chan any, tailedFiles []*tail.Tail) {
	var t T
	file_type := t.GetExportName()
	tailedFiles = make([]*tail.Tail, zbx.DBSyncers+1) // make room for main export
	c = make(chan any, 100)

	//ADD main export file

	for i := 1; i <= zbx.DBSyncers; i++ {
		filename := filepath.Join(zbx.ExportDir, fmt.Sprintf(getBasePath[T](), i))
		tailedFile, err := tail.TailFile(
			filename, tail.Config{
				Follow:        true,
				ReOpen:        true,
				CompleteLines: true,
			})

		tailedFiles[i] = tailedFile
		if err != nil {
			slog.Error("Could not open export", slog.Any("file", filename), slog.Any("error", err))
			return
		}

		go func(filename string, file_index int, file_type string) {
			slog.Info("Opening export file", slog.Any("file", filename))
			labels := prometheus.Labels{
				"file_index":  strconv.Itoa(file_index),
				"export_type": file_type,
			}

			parsedCounter := promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_lines_parsed_total",
				Help:        "The total number of processed lines",
				ConstLabels: labels,
			})
			parsedErrorCounter := promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_lines_invalid_total",
				Help:        "The total number of lines with invalid data",
				ConstLabels: labels,
			})

			slog.Error("Parsing file", slog.Any("file", filename))
			for line := range tailedFiles[file_index].Lines {
				parsed, err := parseLine[T](line)
				parsedCounter.Inc()
				if err != nil {
					parsedErrorCounter.Inc()

					slog.Error("Failed to parse line", slog.Any("file", filename), slog.Any("line_number", line.Num), slog.Any("error", err))
					continue
				}
				c <- parsed
			}
			tailedFiles[file_index].Wait()
		}(filename, i, file_type)
	}

	return
}
