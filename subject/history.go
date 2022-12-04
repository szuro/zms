package subject

import (
	"szuro.net/crapage/observer"
	"szuro.net/crapage/zbx"
)

type Subject interface {
	Register(observer observer.Observer)
	Deregister(observer observer.Observer)
	NotifyAll()
}

type ObserverRegistry map[string]observer.Observer

type HistorySubject struct {
	observers ObserverRegistry
	values    []zbx.History
}

func New() HistorySubject {
	var hs HistorySubject
	hs.observers = make(ObserverRegistry)
	// hs.values = make([]zbx.History, 5)
	return hs
}

func (hs *HistorySubject) Register(observer observer.Observer) {
	hs.observers[observer.GetName()] = observer
}

func (hs *HistorySubject) Degister(observer observer.Observer) {
	delete(hs.observers, observer.GetName())
}

func (hs *HistorySubject) NotifyAll() {
	for _, v := range hs.observers {
		v.SaveHistory(hs.values)
	}
}

func (hs *HistorySubject) AcceptValues(c chan zbx.History) {
	for h := range c {
		hs.values = append(hs.values, h)
		if len(hs.values) >= 2 {
			hs.NotifyAll()
			hs.values = nil
		}
	}
}
