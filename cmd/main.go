package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/oidc-mytoken/utils/httpclient"
	log "github.com/sirupsen/logrus"

	"github.com/zachmann/tip/internal/config"
	"github.com/zachmann/tip/internal/logger"
	"github.com/zachmann/tip/internal/server"
	"github.com/zachmann/tip/internal/version"
)

func main() {
	handleSignals()
	config.Load()
	logger.Init()
	server.Init()
	httpclient.Init("", fmt.Sprintf("tip %s", version.VERSION))
	server.Start()
}

func handleSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGUSR1)
	go func() {
		for {
			sig := <-signals
			switch sig {
			case syscall.SIGHUP:
				reload()
			case syscall.SIGUSR1:
				reloadLogFiles()
			}
		}
	}()
}

func reload() {
	log.Info("Reloading config")
	config.Load()
	logger.SetOutput()
	logger.MustUpdateAccessLogger()
}

func reloadLogFiles() {
	log.Debug("Reloading log files")
	logger.SetOutput()
	logger.MustUpdateAccessLogger()
}
