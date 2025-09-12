package input

import (
	"log/slog"
	"slices"

	"szuro.net/zms/internal/config"
	"szuro.net/zms/internal/logger"
)

type Inputer interface {
	cleanup()
	GetSubjects() map[string]Subjecter
	IsReady() bool
	setFilter()
	Prepare()
	Start()
	Stop() error
}

type baseInput struct {
	config   config.ZMSConf
	subjects map[string]Subjecter
}

func (bs *baseInput) GetSubjects() map[string]Subjecter {
	return bs.subjects
}

func (bs *baseInput) Prepare() {
	bs.setFilter()
	bs.setTargets()
}

func (bs *baseInput) Start() {
	for _, subject := range bs.subjects {
		go subject.AcceptValues()
	}
}

func (bs *baseInput) Stop() error {
	bs.cleanup()
	return nil
}

func (bs *baseInput) cleanup() {
	for _, subject := range bs.subjects {
		subject.Cleanup()
	}
}

func (bs *baseInput) setFilter() {
	for _, subject := range bs.subjects {
		subject.SetFilter(bs.config.TagFilter)
	}
}

func (bs *baseInput) setTargets() {
	for _, target := range bs.config.Targets {
		for name, subject := range bs.subjects {
			if slices.Contains(target.Source, name) {
				t, err := target.ToObserver()
				if err == nil {
					t.InitBuffer(bs.config.WorkingDir, target.OfflineBufferTime)
					subject.Register(t)
				} else {
					logger.Warn("Failed to register target", slog.String("name", t.GetName()))
				}
			}
		}
	}
}
