package subject

import (
	"encoding/binary"
	"log/slog"

	"github.com/nxadm/tail"
	bolt "go.etcd.io/bbolt"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms"
)

type FileInput struct {
	baseInput
	activeTails []*tail.Tail
	indexFinger *bolt.DB
	zbxConf     zbx.ZabbixConf
}

func NewFileInput(zbxConf zbx.ZabbixConf, zmsConf zms.ZMSConf) (fi *FileInput, err error) {
	fi = &FileInput{}
	fi.config = zmsConf
	fi.zbxConf = zbxConf
	fi.subjects = make(map[string]Subjecter)

	fi.indexFinger, err = bolt.Open(zmsConf.PositionIndex, 0600, nil)

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

func (fi *FileInput) Stop() {
	for _, f := range fi.activeTails {
		f.Stop()
		offset, err := f.Tell()
		f.Cleanup()
		if err != nil {
			continue
		}

		fi.indexFinger.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte("offsets"))
			if err != nil {
				return err
			}
			err = bucket.Put([]byte(f.Filename), int64ToBytes(offset))
			if err != nil {
				return err
			}
			return nil

		})
	}
	fi.indexFinger.Close()
	fi.baseInput.Stop()
}

func (fi *FileInput) mkSubjects() {
	zabbix := fi.zbxConf
	var files []*tail.Tail
	for _, v := range zabbix.ExportTypes {
		switch v {
		case zbx.HISTORY:
			hs := NewSubject[zbx.History]()
			hs.Funnel, files = zbx.FileReaderGenerator[zbx.History](zabbix, fi.indexFinger)
			fi.subjects[zbx.HISTORY] = &hs
		case zbx.TREND:
			ts := NewSubject[zbx.Trend]()
			ts.Funnel, files = zbx.FileReaderGenerator[zbx.Trend](zabbix, fi.indexFinger)
			fi.subjects[zbx.TREND] = &ts
		case zbx.EVENT:
			ts := NewSubject[zbx.Event]()
			ts.Funnel, files = zbx.FileReaderGenerator[zbx.Event](zabbix, fi.indexFinger)
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
