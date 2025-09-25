package observer

import (
	"bytes"
	"encoding/gob"
	"log/slog"
	"slices"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/internal/filter"
	"szuro.net/zms/internal/logger"
	zmsplugin "szuro.net/zms/pkg/plugin"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

type Observer interface {
	Cleanup()
	GetName() string
	SetName(name string)
	InitBuffer(path string, t int64)
	SaveHistory(h []zbxpkg.History) bool
	SaveTrends(t []zbxpkg.Trend) bool
	SaveEvents(e []zbxpkg.Event) bool
	SetFilter(filter filter.Filter)
	PrepareMetrics(exports []string)
}

type baseObserver struct {
	name             string
	observerType     string
	monitor          obserwerMetrics
	localFilter      filter.Filter
	offlineBufferTTL time.Duration // Time to keep offline buffer for this observer
	buffer           zmsplugin.ZMSBuffer
	enabledExports   []string // List of enabled export types for this observer
}

// GetName returns the name of the observer.
func (bo *baseObserver) GetName() string {
	return bo.name
}

// SetName sets the name of the baseObserver to the provided string.
func (bo *baseObserver) SetName(name string) {
	bo.name = name
}

func (bo *baseObserver) PrepareMetrics(exports []string) {
	bo.enabledExports = exports
	bo.initObserverMetrics()
}

func (bo *baseObserver) InitBuffer(bufferPath string, t int64) {
	bo.buffer = zmsplugin.ZMSDefaultBuffer{}
	bo.buffer.InitBuffer(bufferPath, t)

}

// Cleanup releases resources held by the baseObserver.
// If offlineBufferTTL is greater than zero, it closes the buffer to free associated resources.
func (bo *baseObserver) Cleanup() {
	if bo.offlineBufferTTL > 0 {
		bo.buffer.Cleanup()
	}
}

func saveToBuffer[T zbxpkg.Export](buffer *badger.DB, toBuffer []T, offlineBufferTTL time.Duration) (err error) {
	var value bytes.Buffer
	enc := gob.NewEncoder(&value)
	txn := buffer.NewTransaction(true)
	for _, item := range toBuffer {
		key := item.Hash()
		enc.Encode(item)
		e := badger.NewEntry(key, value.Bytes()).WithTTL(offlineBufferTTL)
		if err := txn.SetEntry(e); err == badger.ErrTxnTooBig {
			err = txn.Commit()
			if err != nil {
				logger.Error("commit error")
			}
			txn := buffer.NewTransaction(true)
			txn.SetEntry(e)
		}
	}
	err = txn.Commit()
	return
}

func fetchfromBuffer[T zbxpkg.Export](buffer *badger.DB, batchSize int) (buffered []T, err error) {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchSize = batchSize
	txn := buffer.NewTransaction(false)
	defer txn.Discard()
	it := txn.NewIterator(opts)
	defer it.Close()

	decodeFunc := func(val []byte) (T, error) {
		var t T
		dec := gob.NewDecoder(bytes.NewReader(val))
		err := dec.Decode(&t)
		return t, err
	}

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		val, err := item.ValueCopy(nil)
		if err != nil {
			logger.Error("Failed to copy value from buffer", slog.String("observer", buffer.Opts().Dir), slog.Any("error", err))
			continue
		}
		decoded, err := decodeFunc(val)
		if err != nil {
			logger.Error("Failed to decode from buffer", slog.String("observer", buffer.Opts().Dir), slog.Any("error", err))
			continue
		}
		var zero T
		if decoded.GetExportName() == zero.GetExportName() {
			buffered = append(buffered, decoded)
			if len(buffered) >= batchSize {
				break
			}
		}
	}

	return
}

func deleteFromBuffer[T zbxpkg.Export](buffer *badger.DB, buffered []T) (err error) {
	txn := buffer.NewTransaction(true)
	defer txn.Discard()
	for _, item := range buffered {
		key := item.Hash()
		err = txn.Delete(key)
		if err != nil {
			logger.Error("Failed to delete from buffer", slog.String("observer", buffer.Opts().Dir), slog.Any("error", err))
		}
	}
	err = txn.Commit()
	if err != nil {
		logger.Error("Failed to commit", slog.String("observer", buffer.Opts().Dir), slog.Any("error", err))
	}

	return
}

// genericSave is a DRY helper for SaveHistory, SaveTrends, SaveEvents
func genericSave[T zbxpkg.Export](
	items []T,
	filterFunc func(T) bool,
	saveFunc func([]T) ([]T, error),
	buffer *badger.DB,
	offlineBufferTTL time.Duration,
) bool {
	toSave := make([]T, 0, len(items))
	for _, item := range items {
		if filterFunc(item) {
			toSave = append(toSave, item)
		}
	}
	toBuffer, err := saveFunc(toSave)
	if err != nil {
		logger.Error("Failed to save items using saveFunc", slog.Any("error", err))
	}

	if offlineBufferTTL > 0 {
		if err != nil {
			err = saveToBuffer(buffer, toBuffer, offlineBufferTTL)
			if err != nil {
				logger.Error("Failed to save items to offline buffer", slog.Any("error", err))
			}
		} else {
			buffered, _ := fetchfromBuffer[T](buffer, len(toSave))
			if len(buffered) > 0 {
				//delete everything that was fetched from buffer
				//even if some values were NOT successfuly resended
				//depending on the implementation of saveFunc
				//this may lead to data loss or duplicating of some data
				_, err := saveFunc(buffered)
				if err == nil {
					deleteFromBuffer(buffer, buffered)
				} else {
					logger.Error("Failed to re-send buffered items", slog.Any("error", err))
				}
			}
		}
	}
	return true
}

// SaveHistory saves a slice of zbx.History records to the observer's buffer, applying local filtering and serialization.
// It uses a generic saving function with custom filtering, key generation, and serialization/deserialization logic.
// Returns true if the save operation was successful.
func (bo *baseObserver) SaveHistory(h []zbxpkg.History) bool {
	panic("SaveHistory is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveTrends processes and saves a slice of zbx.Trend objects using a generic saving function.
// It applies a local filter to each trend's tags, serializes trends for storage, and manages buffering
// with offline TTL support. Returns true if the save operation succeeds.
func (bo *baseObserver) SaveTrends(t []zbxpkg.Trend) bool {
	panic("SaveTrends is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveEvents processes and saves a slice of zbx.Event objects using a generic saving function.
// It applies a local filter to each event's tags, executes a custom event function, and manages buffering
// with offline TTL support. Events are serialized to and from byte slices for storage.
// Returns true if the events were successfully saved.
func (bo *baseObserver) SaveEvents(e []zbxpkg.Event) bool {
	panic("SaveEvents is not implemented in baseObserver, please implement it in the derived observer type")
}

// SetFilter sets the local filter for the observer.
// The provided filter will be used to determine which events are processed by this observer.
func (bo *baseObserver) SetFilter(filter filter.Filter) {
	bo.localFilter = filter
}

type obserwerMetrics struct {
	historyValuesSent   prometheus.Counter
	historyValuesFailed prometheus.Counter
	trendsValuesSent    prometheus.Counter
	trendsValuesFailed  prometheus.Counter
	eventsValuesSent    prometheus.Counter
	eventsValuesFailed  prometheus.Counter
}

// initObserverMetrics initializes Prometheus counters for tracking shipping operations and errors
// for different export types ("history", "trends", "events") associated with a specific observer.
// It sets up counters for both successful and failed operations, labeling them with the observer's
// type and name, as well as the export type.
//
// Parameters:
//
//	observerType - the type of the observer (used as a label in metrics)
//	name         - the name of the observer (used as a label in metrics)
func (bo *baseObserver) initObserverMetrics() {
	if slices.Contains(bo.enabledExports, zbxpkg.HISTORY) {
		bo.monitor.historyValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": bo.name, "target_type": bo.observerType, "export_type": zbxpkg.HISTORY},
		})

		bo.monitor.historyValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": bo.name, "target_type": bo.observerType, "export_type": zbxpkg.HISTORY},
		})
	}
	if slices.Contains(bo.enabledExports, zbxpkg.TREND) {
		bo.monitor.trendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": bo.name, "target_type": bo.observerType, "export_type": zbxpkg.TREND},
		})

		bo.monitor.trendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": bo.name, "target_type": bo.observerType, "export_type": zbxpkg.TREND},
		})
	}
	if slices.Contains(bo.enabledExports, zbxpkg.EVENT) {
		bo.monitor.eventsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": bo.name, "target_type": bo.observerType, "export_type": zbxpkg.EVENT},
		})

		bo.monitor.eventsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": bo.name, "target_type": bo.observerType, "export_type": zbxpkg.EVENT},
		})
	}
}
