package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/prometheus/model/relabel"
	"gopkg.in/yaml.v2"
)

// http server proxy
// listen at host:port/metrics
//
// 1. listen for incoming proxied scrape requests
// 2. send scrape request to target
// 3. pass metric blob to decoder
// 4. apply relabelling
// 5. return to original proxied request

const (
	listenAddr = ":9090"
	targetAddr = "http://10.33.1.24:9300/metrics"
	configPath = "prometheus.yml"
)

type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

// ScrapeConfig configures a scraping unit for Prometheus.
type ScrapeConfig struct {
	// The job name to which the job label is set by default.
	JobName string `yaml:"job_name"`
	// The HTTP resource path on which to fetch metrics from targets.
	MetricsPath string `yaml:"metrics_path,omitempty"`
	// The URL scheme with which to fetch metrics from targets.
	Scheme string `yaml:"scheme,omitempty"`
	// The hostnames of scrape targets
	StaticConfigs []*StaticConfig `yaml:"static_configs"`
	// List of target relabel configurations.
	RelabelConfigs []*relabel.Config `yaml:"relabel_configs,omitempty"`
	// List of metric relabel configurations.
	MetricRelabelConfigs []*relabel.Config `yaml:"metric_relabel_configs,omitempty"`
}

type Config struct {
	ScrapeConfigs []*ScrapeConfig `yaml:"scrape_configs"`
}

func loadConfig(path string) (*Config, error) {

	c := &Config{}

	file, err := os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("Failed to read config file from %s, err=%w", path, err)
		return nil, err
	}
	err = yaml.Unmarshal(file, c)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshall config file from yaml, err=%w", err)
		return nil, err
	}

	return c, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/metrics", scrape)

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	logger.Log("msg", "serving...")
	go srv.ListenAndServe()

	<-ctx.Done()
	logger.Log("msg", "shutting down...")

	srvShutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := srv.Shutdown(srvShutdownCtx); err != nil {
		err = fmt.Errorf("Failed to shutdown http server: %w", err)
		logger.Log("err", err)
	}
}

func scrapeTarget(path string) ([]byte, error) {

	fmt.Printf("fetching %v\n", path)

	var buf bytes.Buffer

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return buf.Bytes(), err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return buf.Bytes(), err
	}
	defer res.Body.Close()

	if _, err := io.Copy(&buf, res.Body); err != nil {
		return buf.Bytes(), err
	}

	fmt.Printf("done fetching %v\n", path)

	return buf.Bytes(), nil
}

func ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "pong")
}

func scrape(w http.ResponseWriter, r *http.Request) {

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger.Log("msg", "scraping endpoint")

	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Log(err)
	}

	for _, scrapeCfg := range cfg.ScrapeConfigs {
		scheme := scrapeCfg.Scheme
		path := scrapeCfg.MetricsPath
		for _, staticCfg := range scrapeCfg.StaticConfigs {
			for _, target := range staticCfg.Targets {
				url := fmt.Sprintf("%s://%s%s", scheme, target, path)

				buf, err := scrapeTarget(url)
				if err != nil {
					logger.Log("err", err)
				}
				entries, err := parseToSlice(buf, scrapeCfg.RelabelConfigs)
				if err != nil {
					logger.Log("err", err)
				}
				logger.Log("entities", len(entries))
			}
		}
	}
}
