package observer

import (
	"bytes"
	"encoding/gob"
	"path/filepath"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"szuro.net/zms/zbx"
)

func TestSaveToBuffer_Success(t *testing.T) {
	// Register zbx.History for gob
	gob.Register(zbx.History{})

	// Setup temporary directory for BadgerDB
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Prepare test data
	exports := []zbx.History{
		{ItemID: 1, Name: "foo"},
		{ItemID: 2, Name: "bar"},
	}

	ttl := 1 * time.Hour
	err = saveToBuffer(db, exports, ttl)
	if err != nil {
		t.Errorf("saveToBuffer returned error: %v", err)
	}

	// Verify data is in DB
	for _, exp := range exports {
		err := db.View(func(txn *badger.Txn) error {
			key := exp.Hash()
			item, err := txn.Get(key)
			if err != nil {
				t.Errorf("expected key %d not found: %v", exp.ItemID, err)
				return err
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				t.Errorf("failed to get value for key %d: %v", exp.ItemID, err)
				return err
			}
			var decoded zbx.History
			dec := gob.NewDecoder(bytes.NewReader(val))
			if err := dec.Decode(&decoded); err != nil {
				t.Errorf("failed to decode value for key %d: %v", exp.ItemID, err)
			}

			return nil
		})
		if err != nil {
			t.Errorf("db.View failed: %v", err)
		}
	}
}

func TestSaveToBuffer_TxnTooBig(t *testing.T) {
	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb2")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	exports := []zbx.History{
		{ItemID: 1, Name: "foo"},
	}
	ttl := 1 * time.Hour
	err = saveToBuffer(db, exports, ttl)
	if err != nil {
		t.Errorf("saveToBuffer returned error: %v", err)
	}
}

func TestSaveToBuffer_EmptyInput(t *testing.T) {
	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb3")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	var exports []zbx.History
	ttl := 1 * time.Hour
	err = saveToBuffer(db, exports, ttl)
	if err != nil {
		t.Errorf("saveToBuffer returned error for empty input: %v", err)
	}
}

func TestSaveToBuffer_DBClosed(t *testing.T) {
	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb4")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	db.Close() // Close DB before use

	exports := []zbx.History{{ItemID: 1, Name: "foo"}}
	ttl := 1 * time.Hour
	err = saveToBuffer(db, exports, ttl)
	if err == nil {
		t.Errorf("expected error when saving to closed DB, got nil")
	}
}
func TestFetchfromBuffer_Success(t *testing.T) {

	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fetchdb1")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Insert some zbx.History values
	exports := []zbx.History{
		{ItemID: 1, Name: "foo"},
		{ItemID: 2, Name: "bar"},
	}
	for _, exp := range exports {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(exp); err != nil {
			t.Fatalf("failed to encode: %v", err)
		}
		err := db.Update(func(txn *badger.Txn) error {
			key := exp.Hash()
			e := badger.NewEntry(key, buf.Bytes()).WithTTL(1 * time.Hour)
			return txn.SetEntry(e)
		})
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}
	}

	// Fetch from buffer
	result, err := fetchfromBuffer[zbx.History](db, 10)
	if err != nil {
		t.Errorf("fetchfromBuffer returned error: %v", err)
	}
	// To be improved
	if len(exports) != len(result) {
		t.Errorf("got different number of items")
	}
}

func TestFetchfromBuffer_EmptyDB(t *testing.T) {

	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fetchdb2")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	result, err := fetchfromBuffer[zbx.History](db, 10)
	if err != nil {
		t.Errorf("fetchfromBuffer returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestFetchfromBuffer_DecodeError(t *testing.T) {

	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fetchdb3")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Insert invalid gob data
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("badkey"), []byte("notgobdata"))
	})
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	result, err := fetchfromBuffer[zbx.History](db, 10)
	if err != nil {
		t.Errorf("fetchfromBuffer returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result due to decode error, got %v", result)
	}
}

func TestDeleteFromBuffer_Success(t *testing.T) {

	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "deletedb1")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Insert zbx.History values
	exports := []zbx.History{
		{ItemID: 1, Name: "foo"},
		{ItemID: 2, Name: "bar"},
	}
	for _, exp := range exports {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(exp); err != nil {
			t.Fatalf("failed to encode: %v", err)
		}
		err := db.Update(func(txn *badger.Txn) error {
			key := exp.Hash()
			e := badger.NewEntry(key, buf.Bytes()).WithTTL(1 * time.Hour)
			return txn.SetEntry(e)
		})
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}
	}

	// Delete from buffer
	err = deleteFromBuffer(db, exports)
	if err != nil {
		t.Errorf("deleteFromBuffer returned error: %v", err)
	}

	// Verify deletion
	for _, exp := range exports {
		err := db.View(func(txn *badger.Txn) error {
			key := exp.Hash()
			_, err := txn.Get(key)
			if err == nil {
				t.Errorf("expected key %d to be deleted", exp.ItemID)
			}
			if err != badger.ErrKeyNotFound {
				t.Errorf("unexpected error for key %d: %v", exp.ItemID, err)
			}
			return nil
		})
		if err != nil {
			t.Errorf("db.View failed: %v", err)
		}
	}
}

func TestDeleteFromBuffer_EmptyInput(t *testing.T) {

	gob.Register(zbx.History{})

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "deletedb2")
	opts := badger.DefaultOptions(dbPath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	var exports []zbx.History
	err = deleteFromBuffer(db, exports)
	if err != nil {
		t.Errorf("deleteFromBuffer returned error for empty input: %v", err)
	}
}
