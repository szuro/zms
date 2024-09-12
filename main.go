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

var (
	Version, Commit, BuildDate string
)

func printVersionInfo() {
	fmt.Println(fmt.Sprintf("ZMS %s", Version))
	fmt.Println(fmt.Sprintf("Git commit: %s", Commit))
	fmt.Println(fmt.Sprintf("Compilation time: %s", BuildDate))
}

func main() {

	zmsPath := flag.String("c", "/etc/zmsd.yaml", "Path of config file")
	version := flag.Bool("v", false, "Show version info")
	flag.Parse()

	if *version {
		printVersionInfo()
		os.Exit(0)
	}

	zmsConfig := zms.ParseZMSConfig(*zmsPath)
	zbxConfig, _ := zbx.ParseZabbixConfig(zmsConfig.ServerConfig)

	if zbxConfig.ExportDir == "" {
		log.Println("Export not enabled. Aborting.")
		return
	}

	subjects := subject.MkSubjects(zbxConfig, zmsConfig.BufferSize)

	for _, target := range zmsConfig.Targets {
		for name, subject := range subjects {
			if slices.Contains(target.Source, name) {
				t, err := target.ToObserver()
				if err == nil {
					subject.Register(t)
				} else {
					log.Fatalf("Failed ro register: %s", t.GetName())
				}

			}
		}
	}

	http.Handle("/metrics", promhttp.Handler())

	listen := fmt.Sprintf("%s:%d", zmsConfig.Http.ListenAddress, zmsConfig.Http.ListenPort)
	go http.ListenAndServe(listen, nil)

	for delay, isActive := zbx.GetHaStatus(zbxConfig); !isActive; {
		log.Printf("Node is not active, sleeping for %d seconds\n", delay)
		time.Sleep(delay * time.Second)
		delay, isActive = zbx.GetHaStatus(zbxConfig)
	}

	log.Println("Node is active, listing files")

	for _, subject := range subjects {
		subject.SetFilter(zmsConfig.TagFilter)
		go subject.AcceptValues()
	}

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
