package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	proxy "github.com/evanofslack/prom-relabel-proxy"
)

const (
	defaultAddr           = ":9091"
	defaultConfigPath     = "prometheus.yml"
	defaultLogLevel       = "info"
	defaultAppEnv         = "prod"
	serverShutdownTimeout = time.Second * 10
)

func usage() {
	log.Printf("Usage: prom-relabel-proxy [-a listen address] [-c relabel config path] [-e app environment] [-l logLevel]\n")
	flag.PrintDefaults()
}

func parseArgs() (*proxy.Config, error) {
	var addr = flag.String("a", defaultAddr, "address proxy listens on")
	var promConfigPath = flag.String("c", defaultConfigPath, "path to prometheus relabel config")
	var logLevel = flag.String("l", defaultLogLevel, "log level (debug, info, warn, error)")
	var appEnv = flag.String("e", defaultAppEnv, "app environment (debug, prod)")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 0 {
		usage()
		return nil, fmt.Errorf("app does not accept arguments")
	}

	config := proxy.NewConfig(*addr, *promConfigPath, *logLevel, *appEnv)
	return config, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	config, err := parseArgs()
	if err != nil {
		fmt.Println("failed to load config")
		os.Exit(1)
	}

	logger := proxy.NewLogger(config.LogLevel, config.Env)

	promConfig, err := proxy.LoadPromConfig(config.PromConfigPath)
	if err != nil {
		err = fmt.Errorf("failed to load prometheus config, err=%w", err)
		logger.Error(err.Error())
		os.Exit(1)
	}

	scraper := proxy.NewScraper(logger.With("subsystem", "scraper"))
	parser := proxy.NewParser(logger.With("subsystem", "parser"))
	formatter := proxy.NewFormatter(logger.With("subsystem", "formatter"))

	server := proxy.NewServer(logger.With("subsystem", "server"), config.Addr, promConfig, scraper, parser, formatter)
	server.Start()

	<-ctx.Done()

	logger.Info("received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		err = fmt.Errorf("failed to shutdown http server: %w", err)
		logger.Error(err.Error())
	}
}
