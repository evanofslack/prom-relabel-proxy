package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/go-kit/log"
)

const (
	listenAddr = ":9090"
	configPath = "prometheus.yml"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	promConfig, err := loadPromConfig(configPath)
	if err != nil {
		err = fmt.Errorf("Failed to load prometheus config, err=%w", err)
		logger.Log("err", err)
		os.Exit(1)
	}

	scraper := newScraper()
	parser := newParser()
	formatter := newFormatter()

	server := newServer(listenAddr, promConfig, scraper, parser, formatter)
	server.start()
	logger.Log("msg", "serving...")

	<-ctx.Done()
	logger.Log("msg", "shutting down...")

	srvShutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := server.shutdown(srvShutdownCtx); err != nil {
		err = fmt.Errorf("Failed to shutdown http server: %w", err)
		logger.Log("err", err)
	}
}
