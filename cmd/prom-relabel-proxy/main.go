package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	llog "github.com/go-kit/log"

	proxy "github.com/evanofslack/prom-relabel-proxy"
)

const (
	defaultAddr           = ":9091"
	defaultConfigPath     = "prometheus.yml"
	serverShutdownTimeout = time.Second * 10
)

func usage() {
	log.Printf("Usage: prom-relabel-proxy [-a listenAddress] [-c relabelConfigPath]\n")
	flag.PrintDefaults()
}

func parseArgs() (*proxy.Config, error) {
	var addr = flag.String("a", defaultAddr, "Address proxy listens on")
	var promConfigPath = flag.String("c", defaultConfigPath, "Path to prometheus relabel config")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 0 {
		usage()
		return nil, fmt.Errorf("Failed to parse args")
	}

	config := proxy.NewConfig(*addr, *promConfigPath)
	return config, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	logger := llog.NewLogfmtLogger(llog.NewSyncWriter(os.Stderr))
	config, err := parseArgs()
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	promConfig, err := proxy.LoadPromConfig(config.PromConfigPath)
	if err != nil {
		err = fmt.Errorf("Failed to load prometheus config, err=%w", err)
		logger.Log("err", err)
		os.Exit(1)
	}

	scraper := proxy.NewScraper()
	parser := proxy.NewParser()
	formatter := proxy.NewFormatter()

	server := proxy.NewServer(config.Addr, promConfig, scraper, parser, formatter)
	server.Start()
	logger.Log("msg", "serving...")

	<-ctx.Done()
	logger.Log("msg", "shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		err = fmt.Errorf("Failed to shutdown http server: %w", err)
		logger.Log("err", err)
	}
}
