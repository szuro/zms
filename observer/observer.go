package observer

import "szuro.net/zms/zbx"

type Observer interface {
	Cleanup()
	GetName() string
	SetName(name string)
	SaveHistory(h []zbx.History) bool
	SaveTrends(t []zbx.Trend) bool
}

type baseObserver struct {
	name string
}

func (p *baseObserver) GetName() string {
	return p.name
}
func (p *baseObserver) SetName(name string) {
	p.name = name
}

func (p *baseObserver) Cleanup() {

}
