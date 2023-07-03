package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/exp/slices"
	"szuro.net/zms/subject"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms"
)

func main() {

	C := zms.ParseZMSConfig("/etc/zms/zms.yaml")
	c, _ := zbx.ParseZabbixConfig(C.ServerConfig)

	subjects := subject.MkSubjects(c)

	for _, o := range C.Targets {
		for k, v := range subjects {
			if slices.Contains(o.Source, k) {
				v.Register(o.ToObserver())
			}
		}
	}

	for _, v := range subjects {
		v.SetFilter(C.TagFilter)
		go v.AcceptValues()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		switch <-sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			fmt.Print("Exiting...")
			return
		default:
			return
		}
	}

}
