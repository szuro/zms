package input

import (
	"encoding/binary"
	"log/slog"
	"path"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/nxadm/tail"
	"szuro.net/zms/internal/zbx"
	zbxpkg "szuro.net/zms/pkg/zbx"
	"szuro.net/zms/internal/config"
	"szuro.net/zms/internal/logger"
)

type FileInput struct {
	baseInput
	activeTails []*tail.Tail
	fileIndex   *badger.DB // BadgerDB instance for offline buffering
	zbxConf     zbx.ZabbixConf
}

func NewFileInput(zbxConf zbx.ZabbixConf, zmsConf config.ZMSConf) (fi *FileInput, err error) {
	fi = &FileInput{}
	fi.config = zmsConf
	fi.zbxConf = zbxConf
	fi.subjects = make(map[string]Subjecter)

	dbPath := path.Join(zmsConf.WorkingDir, "index.db")
	db, err := badger.Open(badger.DefaultOptions(dbPath).WithLogger(logger.Default()))
	logger.Debug("Initialized BadgerDB for file index", slog.String("path", dbPath))
	if err != nil {
		logger.Error("Failed to open BadgerDB for file index", slog.Any("error", err))
	}
	fi.fileIndex = db

	return
}

func (fi *FileInput) IsReady() bool {
	_, isActive := zbx.GetHaStatus(fi.zbxConf)
	return isActive
}

func (fi *FileInput) Prepare() {
	fi.mkSubjects()
	fi.baseInput.Prepare()
}

func (fi *FileInput) Start() {
	fi.baseInput.Start()
}

func (fi *FileInput) Stop() error {
	for _, f := range fi.activeTails {
		offset, err := f.Tell()
		if err != nil {
			logger.Error("cannot get file offset, resetting to 0", slog.String("file", f.Filename), slog.Any("error", err))
			offset = 0
		}
		f.Stop()
		f.Cleanup()

		err = fi.fileIndex.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte(f.Filename), int64ToBytes(offset))
			return err
		})
		if err != nil {
			logger.Error("error when saving file offset", slog.String("file", f.Filename), slog.Any("error", err))
		}

	}
	fi.fileIndex.Close()
	return fi.baseInput.Stop()

}

func (fi *FileInput) mkSubjects() {
	zabbix := fi.zbxConf
	var files []*tail.Tail
	for _, v := range zabbix.ExportTypes {
		switch v {
		case zbxpkg.HISTORY:
			hs := NewSubject[zbxpkg.History]()
			hs.Funnel, files = zbx.FileReaderGenerator[zbxpkg.History](zabbix, fi.fileIndex, fi.config.BufferSize*2)
			fi.subjects[zbxpkg.HISTORY] = &hs
		case zbxpkg.TREND:
			ts := NewSubject[zbxpkg.Trend]()
			ts.Funnel, files = zbx.FileReaderGenerator[zbxpkg.Trend](zabbix, fi.fileIndex, fi.config.BufferSize*2)
			fi.subjects[zbxpkg.TREND] = &ts
		case zbxpkg.EVENT:
			ts := NewSubject[zbxpkg.Event]()
			ts.Funnel, files = zbx.FileReaderGenerator[zbxpkg.Event](zabbix, fi.fileIndex, fi.config.BufferSize*2)
			fi.subjects[zbxpkg.EVENT] = &ts
		default:
			logger.Error("Export not supported", slog.String("export", v))
		}
	}
	fi.activeTails = append(fi.activeTails, files...)

	for _, subject := range fi.subjects {
		subject.SetBuffer(fi.config.BufferSize)
	}
}

func int64ToBytes(i int64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(i))
	return bytes
}
