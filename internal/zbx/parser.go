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
	"zms.szuro.net/internal/logger"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

func parseHistoryLine(line *tail.Line) (h zbxpkg.History, err error) {
	err = json.Unmarshal([]byte(line.Text), &h)
	if err != nil {
		if h.Type == zbxpkg.FLOAT && h.Value == "" {
			h.Value = "0.0"
		} else if h.Type == zbxpkg.UNSIGNED && h.Value == "" {
			h.Value = "0"
		}
	}
	return
}

func parseTrendLine(line *tail.Line) (t zbxpkg.Trend, err error) {
	err = json.Unmarshal([]byte(line.Text), &t)
	return
}

func parseEventLine(line *tail.Line) (e zbxpkg.Event, err error) {
	err = json.Unmarshal([]byte(line.Text), &e)
	return
}

func parseLine[T zbxpkg.Export](line *tail.Line) (any, error) {
	var t T
	switch any(t).(type) {
	case zbxpkg.History:
		return parseHistoryLine(line)
	case zbxpkg.Trend:
		return parseTrendLine(line)
	case zbxpkg.Event:
		return parseEventLine(line)
	}
	return nil, errors.New("not a supported export type")
}

func getBasePath[T zbxpkg.Export]() (p string) {
	var t T
	switch any(t).(type) {
	case zbxpkg.History:
		p = zbxpkg.HISTORY_EXPORT
	case zbxpkg.Trend:
		p = zbxpkg.TRENDS_EXPORT
	case zbxpkg.Event:
		p = zbxpkg.PROBLEMS_EXPORT
	}

	return
}

func getMainFilePath[T zbxpkg.Export]() (p string) {
	var t T
	switch any(t).(type) {
	case zbxpkg.History:
		p = zbxpkg.HISTORY_MAIN
	case zbxpkg.Trend:
		p = zbxpkg.TRENDS_MAIN
	case zbxpkg.Event:
		p = zbxpkg.PROBLEMS_MAIN
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

func generateFilePaths[T zbxpkg.Export](zbx ZabbixConf) (paths []string) {
	var filename string
	for i := 0; i <= zbx.DBSyncers; i++ {
		if i == 0 {
			filename = getMainFilePath[T]()
		} else {
			filenamePattern := getBasePath[T]()
			filename = fmt.Sprintf(filenamePattern, i)
		}

		paths = append(paths, filepath.Join(zbx.ExportDir, filename))
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

func FileReaderGenerator[T zbxpkg.Export](zbx ZabbixConf, indexDB *badger.DB, chanSize int) (c chan any, tailedFiles []*tail.Tail) {
	var t T
	file_type := t.GetExportName()
	tailedFiles = make([]*tail.Tail, zbx.DBSyncers+1) // make room for main export
	c = make(chan any, chanSize)

	exportFiles := generateFilePaths[T](zbx)
	for i, filename := range exportFiles {
		loc, _ := findLastReadOffset(indexDB, filename)

		tailedFile, err := tail.TailFile(
			filename, tail.Config{
				Follow:        true,
				ReOpen:        true,
				CompleteLines: true,
				Location:      loc,
				Logger:        logger.Default(),
			})

		if err != nil {
			logger.Error("Could not open export", slog.String("file", filename), slog.Any("error", err))
			return
		}
		tailedFiles[i] = tailedFile

		go func(filename string, file_index int, file_type string) {
			parsedCounter, parsedErrorCounter := makeCounters(file_type, file_index)
			logger.Info("Opening and parsing export file", slog.String("file", filename))

			for line := range tailedFiles[file_index].Lines {
				parsed, err := parseLine[T](line)
				parsedCounter.Inc()
				if err != nil {
					parsedErrorCounter.Inc()

					logger.Error("Failed to parse line", slog.String("file", filename), slog.Int("line_number", line.Num), slog.Any("error", err))
					continue
				}
				c <- parsed
			}
			tailedFiles[file_index].Wait()
		}(filename, i, file_type)
	}

	return
}
