package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/exp/slices"
	"szuro.net/zms/subject"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms"

	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	zmsPath := flag.String("c", "/etc/zmsd.yaml", "Path of config file")
	flag.Parse()

	zmsConfig := zms.ParseZMSConfig(*zmsPath)
	zbxConfig, _ := zbx.ParseZabbixConfig(zmsConfig.ServerConfig)

	if zbxConfig.ExportDir == "" {
		log.Println("Export not enabled. Aborting.")
		return
	}

	for delay, isActive := zbx.GetHaStatus(zbxConfig); !isActive; {
		log.Printf("Node is not active, sleeping for %d seconds\n", delay)
		time.Sleep(delay * time.Second)
		delay, isActive = zbx.GetHaStatus(zbxConfig)
	}

	log.Println("Node is active, listing files")

	subjects := subject.MkSubjects(zbxConfig, zmsConfig.BufferSize)

	for _, target := range zmsConfig.Targets {
		for name, subject := range subjects {
			if slices.Contains(target.Source, name) {
				subject.Register(target.ToObserver())
			}
		}
	}

	for _, subject := range subjects {
		subject.SetFilter(zmsConfig.TagFilter)
		go subject.AcceptValues()
	}

	http.Handle("/metrics", promhttp.Handler())

	listen := fmt.Sprintf("%s:%d", zmsConfig.Http.ListenAddress, zmsConfig.Http.ListenPort)
	http.ListenAndServe(listen, nil)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		switch <-sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:

			for _, subject := range subjects {
				subject.Cleanup()
			}

			fmt.Print("Exiting...")
			return
		default:
			return
		}
	}

}
