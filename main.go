package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	fmt.Printf("ZMS %s\n", Version)
	fmt.Printf("Git commit: %s\n", Commit)
	fmt.Printf("Compilation time: %s\n", BuildDate)
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
	slog.SetLogLoggerLevel(zmsConfig.GetLogLevel())

	var input subject.Inputer

	switch zmsConfig.Mode {
	case zms.FILE_MODE:
		zbxConfig, _ := zbx.ParseZabbixConfig(zmsConfig.ServerConfig)
		if zbxConfig.ExportDir == "" {
			slog.Error("Export not enabled. Aborting.")
			return
		}
		input, _ = subject.NewFileInput(zbxConfig, zmsConfig)
	case zms.HTTP_MODE:
		panic("HTTP mode is not implemented yet")
		// input, _ = subject.NewHTTPInput(zmsConfig)
	default:
		slog.Error("Unknown mode", slog.String("mode", zmsConfig.Mode))
		os.Exit(1)
	}

	input.Prepare()

	http.Handle("/metrics", promhttp.Handler())

	listen := fmt.Sprintf("%s:%d", zmsConfig.Http.ListenAddress, zmsConfig.Http.ListenPort)
	go http.ListenAndServe(listen, nil)

	for isActive := input.IsReady(); !isActive; {
		delay := time.Duration(zbx.DEFAULT_DELAY)
		slog.Info("Input is not active, sleeping for ", slog.Any("delay", delay))
		time.Sleep(delay)
		isActive = input.IsReady()
	}

	slog.Info("Input is active")

	input.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		switch <-sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			input.Stop()

			slog.Info("Exiting...")
			return
		default:
			return
		}
	}

}
