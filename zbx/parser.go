package zbx

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	badger "github.com/dgraph-io/badger/v4"

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

func getMainFilePath[T Export]() (p string) {
	var t T
	switch any(t).(type) {
	case History:
		p = HISTORY_MAIN
	case Trend:
		p = TRENDS_MAIN
	case Event:
		p = PROBLEMS_MAIN
	}

	return
}

func bytesToInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func findLastReadOffset(indexDB *badger.DB, filename string) (location *tail.SeekInfo, err error) {
	location = &tail.SeekInfo{}
	location.Whence = io.SeekStart

	err = indexDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(filename))
		if item != nil {
			err = item.Value(func(val []byte) error {
				offset := bytesToInt64(val)
				location.Offset = offset
				return nil
			})
		}
		if err != nil {
			location.Offset = 0
		}

		return err
	})

	f, err := os.Stat(filename)
	// offset greater than size means the file was rotated
	if err != nil || location.Offset > f.Size() {
		location.Offset = 0
	}

	return
}

func generateFilePaths[T Export](zbx ZabbixConf) (paths []string) {
	var filename string
	for i := 0; i <= zbx.DBSyncers; i++ {
		if i == 0 {
			filename = getMainFilePath[T]()
		} else {
			filenamePattern := getBasePath[T]()
			filename = filepath.Join(zbx.ExportDir, fmt.Sprintf(filenamePattern, i))
		}

		paths = append(paths, filename)
	}
	return
}

func makeCounters(file_type string, file_index int) (parsedCounter, parsedErrorCounter prometheus.Counter) {
	labels := prometheus.Labels{
		"file_index":  strconv.Itoa(file_index),
		"export_type": file_type,
	}

	parsedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_lines_parsed_total",
		Help:        "The total number of processed lines",
		ConstLabels: labels,
	})
	parsedErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_lines_invalid_total",
		Help:        "The total number of lines with invalid data",
		ConstLabels: labels,
	})
	return
}

func FileReaderGenerator[T Export](zbx ZabbixConf, indexDB *badger.DB) (c chan any, tailedFiles []*tail.Tail) {
	var t T
	file_type := t.GetExportName()
	tailedFiles = make([]*tail.Tail, zbx.DBSyncers+1) // make room for main export
	c = make(chan any, 100)

	exportFiles := generateFilePaths[T](zbx)
	for i, filename := range exportFiles {
		loc, _ := findLastReadOffset(indexDB, filename)

		tailedFile, err := tail.TailFile(
			filename, tail.Config{
				Follow:        true,
				ReOpen:        true,
				CompleteLines: true,
				Location:      loc,
			})

		if err != nil {
			slog.Error("Could not open export", slog.Any("file", filename), slog.Any("error", err))
			return
		}
		tailedFiles[i] = tailedFile

		go func(filename string, file_index int, file_type string) {
			parsedCounter, parsedErrorCounter := makeCounters(file_type, file_index)
			slog.Info("Opening and parsing export file", slog.Any("file", filename))

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
