package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"szuro.net/zms/internal/config"
	"szuro.net/zms/internal/input"
	"szuro.net/zms/internal/logger"
	"szuro.net/zms/internal/plugin"
	"szuro.net/zms/internal/zbx"

	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func printVersionInfo() {
	fmt.Printf("ZMS %s\n", config.Version)
	fmt.Printf("Git commit: %s\n", config.Commit)
	fmt.Printf("Compilation time: %s\n", config.BuildDate)
}

func main() {

	zmsPath := flag.String("c", "/etc/zmsd.yaml", "Path of config file")
	version := flag.Bool("v", false, "Show version info")
	flag.Parse()

	if *version {
		printVersionInfo()
		os.Exit(0)
	}

	zmsConfig := config.ParseZMSConfig(*zmsPath)
	logger.SetLogLevel(zmsConfig.GetLogLevel())

	// Load plugins if plugin directory is configured
	if zmsConfig.PluginsDir != "" {
		logger.Info("Loading plugins", slog.String("dir", zmsConfig.PluginsDir))
		if err := plugin.GetRegistry().LoadPluginsFromDir(zmsConfig.PluginsDir); err != nil {
			logger.Error("Failed to load plugins", slog.Any("error", err))
			// Continue execution - plugins are optional
		}

		// List loaded plugins
		plugins := plugin.GetRegistry().ListPlugins()
		for _, p := range plugins {
			logger.Info("Loaded plugin",
				slog.String("name", p.Name),
				slog.String("version", p.Version))
		}
	}

	var inp input.Inputer

	switch zmsConfig.Mode {
	case config.FILE_MODE:
		zbxConfig, _ := zbx.ParseZabbixConfig(zmsConfig.ServerConfig)
		if zbxConfig.ExportDir == "" {
			logger.Error("Export not enabled. Aborting.")
			return
		}
		inp, _ = input.NewFileInput(zbxConfig, zmsConfig)
	case config.HTTP_MODE:
		inp, _ = input.NewHTTPInput(zmsConfig)
	default:
		logger.Error("Unknown mode", slog.String("mode", zmsConfig.Mode))
		os.Exit(1)
	}

	inp.Prepare()
	config.ZmsInfo.Set(1)

	http.Handle("/metrics", promhttp.Handler())

	listen := fmt.Sprintf("%s:%d", zmsConfig.Http.ListenAddress, zmsConfig.Http.ListenPort)
	go http.ListenAndServe(listen, nil)

	for isActive := inp.IsReady(); !isActive; {
		delay := time.Duration(zbx.DEFAULT_DELAY)
		logger.Info("Input is not active, sleeping for ", slog.Duration("delay", delay))
		time.Sleep(delay)
		isActive = inp.IsReady()
	}

	logger.Info("Input is active")

	inp.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		switch <-sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			err := inp.Stop()
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
