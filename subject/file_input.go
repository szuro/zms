package subject

import (
	"encoding/binary"
	"log/slog"
	"os"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/nxadm/tail"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms"
)

type FileInput struct {
	baseInput
	activeTails []*tail.Tail
	fileIndex   *badger.DB // BadgerDB instance for offline buffering
	zbxConf     zbx.ZabbixConf
}

func NewFileInput(zbxConf zbx.ZabbixConf, zmsConf zms.ZMSConf) (fi *FileInput, err error) {
	fi = &FileInput{}
	fi.config = zmsConf
	fi.zbxConf = zbxConf
	fi.subjects = make(map[string]Subjecter)
	db, err := badger.Open(badger.DefaultOptions(
		zmsConf.WorkingDir + string(os.PathSeparator) + "index.db",
	))
	if err != nil {
		slog.Error("Failed to open BadgerDB for offline buffering", "error", err)
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
			slog.Error("cannot get file offset, resetting to 0", slog.Any("file", f.Filename), slog.Any("error", err))
			offset = 0
		}
		f.Stop()
		f.Cleanup()

		err = fi.fileIndex.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte(f.Filename), int64ToBytes(offset))
			return err
		})
		if err != nil {
			slog.Error("error when saving file offset", slog.Any("file", f.Filename), slog.Any("error", err))
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
		case zbx.HISTORY:
			hs := NewSubject[zbx.History]()
			hs.Funnel, files = zbx.FileReaderGenerator[zbx.History](zabbix, fi.fileIndex, fi.config.BufferSize*2)
			fi.subjects[zbx.HISTORY] = &hs
		case zbx.TREND:
			ts := NewSubject[zbx.Trend]()
			ts.Funnel, files = zbx.FileReaderGenerator[zbx.Trend](zabbix, fi.fileIndex, fi.config.BufferSize*2)
			fi.subjects[zbx.TREND] = &ts
		case zbx.EVENT:
			ts := NewSubject[zbx.Event]()
			ts.Funnel, files = zbx.FileReaderGenerator[zbx.Event](zabbix, fi.fileIndex, fi.config.BufferSize*2)
			fi.subjects[zbx.EVENT] = &ts
		default:
			slog.Error("Export not supported", slog.Any("export", v))
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
