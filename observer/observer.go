package observer

import "szuro.net/crapage/zbx"

type Observer interface {
	GetName() string
	SetName(name string)
	SaveHistory(h []zbx.History) bool
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
