package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-kit/log"
)

type Server struct {
	promConfig *PromConfig
	srv        *http.Server
	scraper    *Scraper
	parser     *Parser
	formatter  *Formatter
}

func NewServer(addr string, promConfig *PromConfig, scraper *Scraper, parser *Parser, formatter *Formatter) *Server {

	srv := &http.Server{
		Addr: addr,
	}

	s := &Server{
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

func (s *Server) Start() {
	go s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "pong")
}

func (s *Server) scrape(w http.ResponseWriter, r *http.Request) {

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
					// TODO: write error to w
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
