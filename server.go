package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-kit/log"
)

type server struct {
	promConfig *promConfig
	srv        *http.Server
	scraper    *scraper
	parser     *parser
	formatter  *formatter
}

func newServer(addr string, promConfig *promConfig, scraper *scraper, parser *parser, formatter *formatter) *server {

	srv := &http.Server{
		Addr: listenAddr,
	}

	s := &server{
		promConfig: promConfig,
		srv:        srv,
		scraper:    scraper,
		parser:     parser,
		formatter:  formatter,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", s.ping)
	mux.HandleFunc("/metrics", s.scrape)

	s.srv.Handler = mux

	return s
}

func (s *server) start() {
	go s.srv.ListenAndServe()
}

func (s *server) shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *server) ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "pong")
}

func (s *server) scrape(w http.ResponseWriter, r *http.Request) {

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger.Log("msg", "scraping endpoint")

	for _, scrapeCfg := range s.promConfig.ScrapeConfigs {
		scheme := scrapeCfg.Scheme
		path := scrapeCfg.MetricsPath
		for _, staticCfg := range scrapeCfg.StaticConfigs {
			for _, target := range staticCfg.Targets {

				// Scrape metrics from proxied target
				url := fmt.Sprintf("%s://%s%s", scheme, target, path)
				buf, err := s.scraper.scrape(url)
				if err != nil {
					logger.Log("err", err)
				}

				// Parse proxied metrics
				entries, err := s.parser.parse(buf, scrapeCfg.RelabelConfigs)
				if err != nil {
					logger.Log("err", err)
				}

				// Format proxied metrics
				//TODO: aggregate all entries before final resp
				output := s.formatter.format(entries)
				io.WriteString(w, output)
			}
		}
	}
}
