package plugin

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log/slog"
	"path"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"szuro.net/zms/internal/logger"
	"szuro.net/zms/pkg/zbx"
)

// ZMSBuffer defines the interface for offline data buffering.
// This interface provides persistent storage for Zabbix export data when the target
// destination is temporarily unavailable, ensuring data reliability and delivery guarantees.
//
// The buffer uses BadgerDB for local persistence and supports TTL-based expiration
// to prevent unbounded storage growth. Data is serialized using gob encoding.
type ZMSBuffer interface {
	// History buffer operations

	// BufferHistory stores history records in the offline buffer.
	// Used when the target destination is unavailable.
	BufferHistory(history []zbx.History) (err error)

	// FetchHistory retrieves a specified number of history records from the buffer.
	// Returns up to 'number' records, or fewer if not enough are available.
	FetchHistory(number int) (history []zbx.History, err error)

	// DeleteHistory removes the specified history records from the buffer.
	// Used to clean up successfully processed records.
	DeleteHistory(history []zbx.History) (err error)

	// Trends buffer operations

	// BufferTrends stores trend records in the offline buffer.
	// Used when the target destination is unavailable.
	BufferTrends(trends []zbx.Trend) (err error)

	// FetchTrends retrieves a specified number of trend records from the buffer.
	// Returns up to 'number' records, or fewer if not enough are available.
	FetchTrends(number int) (trends []zbx.Trend, err error)

	// DeleteTrends removes the specified trend records from the buffer.
	// Used to clean up successfully processed records.
	DeleteTrends(trends []zbx.Trend) (err error)

	// Events buffer operations

	// BufferEvents stores event records in the offline buffer.
	// Used when the target destination is unavailable.
	BufferEvents(events []zbx.Event) (err error)

	// FetchEvents retrieves a specified number of event records from the buffer.
	// Returns up to 'number' records, or fewer if not enough are available.
	FetchEvents(number int) (events []zbx.Event, err error)

	// DeleteEvents removes the specified event records from the buffer.
	// Used to clean up successfully processed records.
	DeleteEvents(events []zbx.Event) (err error)

	// Buffer management

	// InitBuffer initializes the buffer with the specified path and TTL.
	// bufferPath: directory where buffer data will be stored
	// t: time-to-live in hours for buffered records
	InitBuffer(bufferPath string, t int64)

	// Cleanup releases buffer resources and closes the database connection.
	Cleanup()
}

// ZMSDefaultBuffer is the default implementation of ZMSBuffer using BadgerDB.
// It provides persistent, transactional storage with configurable TTL for automatic cleanup.
// Data is stored using gob encoding and indexed by hash keys generated from export records.
type ZMSDefaultBuffer struct {
	// offlineBufferTTL defines how long records are kept in the buffer before expiration
	offlineBufferTTL time.Duration

	// bufferPath is the directory path where BadgerDB files are stored
	bufferPath string

	// buffer is the BadgerDB instance used for persistent storage
	buffer *badger.DB
}

// InitBuffer initializes the BadgerDB buffer with the specified path and TTL.
// If TTL is 0, buffering is disabled. Otherwise, a BadgerDB instance is created
// at the specified path with automatic TTL-based cleanup.
func (b ZMSDefaultBuffer) InitBuffer(bufferPath string, ttl int64) {
	b.offlineBufferTTL = time.Duration(ttl) * time.Hour
	if b.offlineBufferTTL > 0 {
		db, err := badger.Open(badger.DefaultOptions(
			path.Join(b.bufferPath),
		).WithLogger(logger.Default()))
		logger.Debug("Initialized BadgerDB for offline buffering", slog.String("path", path.Join(b.bufferPath)))
		if err != nil {
			logger.Error("Failed to open BadgerDB for offline buffering", slog.Any("error", err))
		}
		b.buffer = db
	}
}

// Cleanup releases resources held by the baseObserver.
// If offlineBufferTTL is greater than zero, it closes the buffer to free associated resources.
func (b ZMSDefaultBuffer) Cleanup() {
	if b.buffer != nil {
		b.buffer.Close()
	}
}

func (b ZMSDefaultBuffer) BufferHistory(history []zbx.History) (err error) {
	return saveToBuffer[zbx.History](b.buffer, history, b.offlineBufferTTL)
}
func (b ZMSDefaultBuffer) FetchHistory(number int) (history []zbx.History, err error) {
	return fetchfromBuffer[zbx.History](b.buffer, number)
}
func (b ZMSDefaultBuffer) DeleteHistory(history []zbx.History) (err error) {
	return deleteFromBuffer[zbx.History](b.buffer, history)
}
func (b ZMSDefaultBuffer) BufferTrends(trends []zbx.Trend) (err error) {
	return saveToBuffer(b.buffer, trends, b.offlineBufferTTL)
}
func (b ZMSDefaultBuffer) FetchTrends(number int) (trends []zbx.Trend, err error) {
	return fetchfromBuffer[zbx.Trend](b.buffer, number)
}
func (b ZMSDefaultBuffer) DeleteTrends(trends []zbx.Trend) (err error) {
	return deleteFromBuffer[zbx.Trend](b.buffer, trends)
}
func (b ZMSDefaultBuffer) BufferEvents(events []zbx.Event) (err error) {
	return saveToBuffer(b.buffer, events, b.offlineBufferTTL)
}
func (b ZMSDefaultBuffer) FetchEvents(number int) (events []zbx.Event, err error) {
	return fetchfromBuffer[zbx.Event](b.buffer, number)
}
func (b ZMSDefaultBuffer) DeleteEvents(events []zbx.Event) (err error) {
	return deleteFromBuffer[zbx.Event](b.buffer, events)
}

func saveToBuffer[T zbx.Export](buffer *badger.DB, toBuffer []T, offlineBufferTTL time.Duration) (err error) {
	if buffer == nil {
		return errors.New("Cannot write to nil buffer")
	}
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

func fetchfromBuffer[T zbx.Export](buffer *badger.DB, batchSize int) (buffered []T, err error) {
	if buffer == nil {
		return buffered, errors.New("Cannot read from nil buffer")
	}
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

func deleteFromBuffer[T zbx.Export](buffer *badger.DB, buffered []T) (err error) {
	if buffer == nil {
		return errors.New("Cannot delete from nil buffer")
	}
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
