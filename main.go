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
	"szuro.net/zms/zms/logger"

	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func printVersionInfo() {
	fmt.Printf("ZMS %s\n", zms.Version)
	fmt.Printf("Git commit: %s\n", zms.Commit)
	fmt.Printf("Compilation time: %s\n", zms.BuildDate)
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
	logger.SetLogLevel(zmsConfig.GetLogLevel())

	var input subject.Inputer

	switch zmsConfig.Mode {
	case zms.FILE_MODE:
		zbxConfig, _ := zbx.ParseZabbixConfig(zmsConfig.ServerConfig)
		if zbxConfig.ExportDir == "" {
			logger.Error("Export not enabled. Aborting.")
			return
		}
		input, _ = subject.NewFileInput(zbxConfig, zmsConfig)
	case zms.HTTP_MODE:
		input, _ = subject.NewHTTPInput(zmsConfig)
	default:
		logger.Error("Unknown mode", slog.String("mode", zmsConfig.Mode))
		os.Exit(1)
	}

	input.Prepare()
	zms.ZmsInfo.Set(1)

	http.Handle("/metrics", promhttp.Handler())

	listen := fmt.Sprintf("%s:%d", zmsConfig.Http.ListenAddress, zmsConfig.Http.ListenPort)
	go http.ListenAndServe(listen, nil)

	for isActive := input.IsReady(); !isActive; {
		delay := time.Duration(zbx.DEFAULT_DELAY)
		logger.Info("Input is not active, sleeping for ", slog.Duration("delay", delay))
		time.Sleep(delay)
		isActive = input.IsReady()
	}

	logger.Info("Input is active")

	input.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		switch <-sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			err := input.Stop()
			if err != nil {
				logger.Error("stopping failed", slog.Any("error", err))
			}
			logger.Info("Exiting...")
			return
		default:
			return
		}
	}

}
