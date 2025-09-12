package zbx

import (
	"io"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

// Helper to create a temp file with given size
func createTempFileWithSize(t *testing.T, size int64) (string, func()) {
	tmpfile, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	defer tmpfile.Close()
	if size > 0 {
		_, err = tmpfile.Seek(size-1, io.SeekStart)
		require.NoError(t, err)
		_, err = tmpfile.Write([]byte{0})
		require.NoError(t, err)
	}
	return tmpfile.Name(), func() { os.Remove(tmpfile.Name()) }
}

// Helper to create a Badger DB in a temp dir
func createTempBadgerDB(t *testing.T) (*badger.DB, func()) {
	dir := t.TempDir()
	opts := badger.DefaultOptions(dir).WithLogger(nil)
	db, err := badger.Open(opts)
	require.NoError(t, err)
	return db, func() { db.Close() }
}

func TestFindLastReadOffset_NoOffsetInDB(t *testing.T) {
	db, cleanup := createTempBadgerDB(t)
	defer cleanup()
	filename, remove := createTempFileWithSize(t, 100)
	defer remove()

	location, err := findLastReadOffset(db, filename)
	require.NoError(t, err)
	require.NotNil(t, location)
	require.Equal(t, io.SeekStart, location.Whence)
	require.Equal(t, int64(0), location.Offset)
}

func TestFindLastReadOffset_OffsetInDBWithinFileSize(t *testing.T) {
	db, cleanup := createTempBadgerDB(t)
	defer cleanup()
	filename, remove := createTempFileWithSize(t, 200)
	defer remove()

	// Write offset 100 to DB
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(filename), []byte{0, 0, 0, 0, 0, 0, 0, 100})
	})
	require.NoError(t, err)

	location, err := findLastReadOffset(db, filename)
	require.NoError(t, err)
	require.NotNil(t, location)
	require.Equal(t, io.SeekStart, location.Whence)
	require.Equal(t, int64(100), location.Offset)
}

func TestFindLastReadOffset_OffsetInDBGreaterThanFileSize(t *testing.T) {
	db, cleanup := createTempBadgerDB(t)
	defer cleanup()
	filename, remove := createTempFileWithSize(t, 50)
	defer remove()

	// Write offset 100 to DB (greater than file size)
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(filename), []byte{0, 0, 0, 0, 0, 0, 0, 100})
	})
	require.NoError(t, err)

	location, err := findLastReadOffset(db, filename)
	require.NoError(t, err)
	require.NotNil(t, location)
	require.Equal(t, io.SeekStart, location.Whence)
	require.Equal(t, int64(0), location.Offset)
}

func TestFindLastReadOffset_FileDoesNotExist(t *testing.T) {
	db, cleanup := createTempBadgerDB(t)
	defer cleanup()
	filename := "nonexistent_file_123456"

	_, err := findLastReadOffset(db, filename)
	require.Error(t, err)
}
