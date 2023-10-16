package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type Server struct {
	logger     *slog.Logger
	promConfig *PromConfig
	srv        *http.Server
	scraper    *Scraper
	parser     *Parser
	formatter  *Formatter
}

func NewServer(logger *slog.Logger, addr, metricsPath string, promConfig *PromConfig, scraper *Scraper, parser *Parser, formatter *Formatter) *Server {

	srv := &http.Server{
		Addr: addr,
	}

	s := &Server{
		logger:     logger,
		promConfig: promConfig,
		srv:        srv,
		scraper:    scraper,
		parser:     parser,
		formatter:  formatter,
	}

	if !strings.HasPrefix(metricsPath, "/") {
		metricsPath = fmt.Sprintf("/%s", metricsPath)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", s.ready)
	mux.HandleFunc("/healthy", s.healthy)
	mux.HandleFunc(metricsPath, s.scrape)

	s.srv.Handler = mux

	return s
}

func (s *Server) Start() {
	s.logger.Info(fmt.Sprintf("starting server at addr %s", s.srv.Addr))
	go s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down")
	return s.srv.Shutdown(ctx)
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "ready")
}

func (s *Server) healthy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "healthy")
}

func (s *Server) scrape(w http.ResponseWriter, r *http.Request) {

	logger := s.logger.With("endpoint", "scrape")
	logger.Debug("handling endpoint")

	out := ""
	scrapes, failedScrapes := 0, 0

	for _, scrapeCfg := range s.promConfig.ScrapeConfigs {
		scheme := scrapeCfg.Scheme
		path := scrapeCfg.MetricsPath
		for _, staticCfg := range scrapeCfg.StaticConfigs {
			for _, target := range staticCfg.Targets {

				// Scrape metrics from proxied target
				url := fmt.Sprintf("%s://%s%s", scheme, target, path)
				buf, err := s.scraper.scrape(url, r)
				scrapes++
				if err != nil {
					failedScrapes++
					logger.Error(err.Error())
				}

				// Parse proxied metrics
				entries, err := s.parser.parse(buf, scrapeCfg.RelabelConfigs)
				if err != nil {
					logger.Error(err.Error())
				}

				// Format proxied metrics
				output := s.formatter.format(entries)
				out += output
			}
		}
	}

	if failedScrapes != 0 {
		logger.Warn(fmt.Sprintf("failed scraping %d/%d targets", failedScrapes, scrapes))
		// If every scrape failed, return error code
		if scrapes == failedScrapes {
			logger.Warn("All scrapes failed, responding with error code")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// If at least one scrape succeeds, return the metrics
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, out)
	logger.Debug(fmt.Sprintf("finished scraping %d targets", scrapes))
}
