package observer

import (
	"bytes"
	"encoding/gob"
	"log/slog"
	"os"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms/filter"
)

type Observer interface {
	Cleanup()
	GetName() string
	SetName(name string)
	InitBuffer(path string, t int64)
	SaveHistory(h []zbx.History) bool
	SaveTrends(t []zbx.Trend) bool
	SaveEvents(e []zbx.Event) bool
	SetFilter(filter filter.Filter)
}

type baseObserver struct {
	name             string
	monitor          obserwerMetrics
	localFilter      filter.Filter
	offlineBufferTTL time.Duration // Time to keep offline buffer for this observer
	buffer           *badger.DB    // BadgerDB instance for offline buffering
	workingDir       string        // Directory for storing local data
}

// GetName returns the name of the observer.
func (p *baseObserver) GetName() string {
	return p.name
}

// SetName sets the name of the baseObserver to the provided string.
func (p *baseObserver) SetName(name string) {
	p.name = name
}

func (p *baseObserver) InitBuffer(path string, t int64) {
	p.offlineBufferTTL = time.Duration(t) * time.Hour
	if p.offlineBufferTTL > 0 {
		db, err := badger.Open(badger.DefaultOptions(
			path + string(os.PathSeparator) + p.name + ".db",
		))
		if err != nil {
			slog.Error("Failed to open BadgerDB for offline buffering", "error", err)
		}
		p.buffer = db
	}
}

// Cleanup releases resources held by the baseObserver.
// If offlineBufferTTL is greater than zero, it closes the buffer to free associated resources.
func (p *baseObserver) Cleanup() {
	if p.offlineBufferTTL > 0 {
		p.buffer.Close()
	}
}

// genericSave is a DRY helper for SaveHistory, SaveTrends, SaveEvents
func genericSave[T any](
	items []T,
	filterFunc func(T) bool,
	saveFunc func([]T) ([]T, error),
	buffer *badger.DB,
	offlineBufferTTL time.Duration,
	keyFunc func(T) []byte,
	decodeFunc func([]byte) (T, error),
) bool {
	toSave := make([]T, 0, len(items))
	for _, item := range items {
		if filterFunc(item) {
			toSave = append(toSave, item)
		}
	}
	toBuffer, err := saveFunc(toSave)

	if offlineBufferTTL > 0 {
		if err != nil {
			var value bytes.Buffer
			enc := gob.NewEncoder(&value)
			txn := buffer.NewTransaction(true)
			for _, item := range toBuffer {
				key := keyFunc(item)
				enc.Encode(item)
				e := badger.NewEntry(key, value.Bytes()).WithTTL(offlineBufferTTL)
				if err := txn.SetEntry(e); err == badger.ErrTxnTooBig {
					_ = txn.Commit()
					txn := buffer.NewTransaction(true)
					txn.SetEntry(e)
				}
			}
			_ = txn.Commit()
		} else if err == nil {
			fetchSize := len(toSave)
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = fetchSize
			txn := buffer.NewTransaction(false)
			defer txn.Discard()
			it := txn.NewIterator(opts)
			defer it.Close()

			var buffered []T
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				val, err := item.ValueCopy(nil)
				if err != nil {
					continue
				}
				decoded, err := decodeFunc(val)
				if err != nil {
					continue
				}
				buffered = append(buffered, decoded)
			}

			if len(buffered) > 0 {
				_, err := saveFunc(buffered)
				if err == nil {
					// Remove successfully processed items from buffer
					txn := buffer.NewTransaction(true)
					defer txn.Discard()
					for _, item := range buffered {
						key := keyFunc(item)
						_ = txn.Delete(key)
					}
					_ = txn.Commit()
				}
			}
		}
	}
	return true
}

// SaveHistory saves a slice of zbx.History records to the observer's buffer, applying local filtering and serialization.
// It uses a generic saving function with custom filtering, key generation, and serialization/deserialization logic.
// Returns true if the save operation was successful.
func (p *baseObserver) SaveHistory(h []zbx.History) bool {
	panic("SaveHistory is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveTrends processes and saves a slice of zbx.Trend objects using a generic saving function.
// It applies a local filter to each trend's tags, serializes trends for storage, and manages buffering
// with offline TTL support. Returns true if the save operation succeeds.
func (p *baseObserver) SaveTrends(t []zbx.Trend) bool {
	panic("SaveTrends is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveEvents processes and saves a slice of zbx.Event objects using a generic saving function.
// It applies a local filter to each event's tags, executes a custom event function, and manages buffering
// with offline TTL support. Events are serialized to and from byte slices for storage.
// Returns true if the events were successfully saved.
func (p *baseObserver) SaveEvents(e []zbx.Event) bool {
	panic("SaveEvents is not implemented in baseObserver, please implement it in the derived observer type")
}

// SetFilter sets the local filter for the observer.
// The provided filter will be used to determine which events are processed by this observer.
func (p *baseObserver) SetFilter(filter filter.Filter) {
	p.localFilter = filter
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
func (m *obserwerMetrics) initObserverMetrics(observerType, name string) {
	m.historyValuesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_operations_total",
		Help:        "Total number of shipping operations",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "history"},
	})

	m.historyValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "Total number of shipping errors",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "history"},
	})

	m.trendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_operations_total",
		Help:        "Total number of shipping operations",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "trends"},
	})

	m.trendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "Total number of shipping errors",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "trends"},
	})

	m.eventsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_operations_total",
		Help:        "Total number of shipping operations",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "events"},
	})

	m.eventsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "Total number of shipping errors",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "events"},
	})
}
