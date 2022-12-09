package subject

import (
	"szuro.net/crapage/observer"
	"szuro.net/crapage/zbx"
)

type Subjecter interface {
	AcceptValues()
	Register(observer observer.Observer)
	Deregister(observer observer.Observer)
	NotifyAll()
}

type ObserverRegistry map[string]observer.Observer

type Subject[T zbx.Export] struct {
	observers ObserverRegistry
	values    []T
	buffer    int
	Funnel    chan any
}

func NewSubject[t zbx.Export]() (s Subject[t]) {
	s.observers = make(ObserverRegistry)
	s.buffer = 2
	return s
}

func (bs *Subject[T]) Register(observer observer.Observer) {
	bs.observers[observer.GetName()] = observer
}

func (bs *Subject[T]) Deregister(observer observer.Observer) {
	delete(bs.observers, observer.GetName())
}

func (bs *Subject[T]) NotifyAll() {
	var t T
	for _, v := range bs.observers {
		switch any(t).(type) {
		case zbx.History:
			h := any(bs.values).([]zbx.History)
			v.SaveHistory(h)
		}
	}
}
func (bs *Subject[T]) AcceptValues() {
	for h := range bs.Funnel {
		v := h.(T)
		bs.values = append(bs.values, v)
		if len(bs.values) >= bs.buffer {
			bs.NotifyAll()
			bs.values = nil
		}
	}
}