package proxy

import (
	"fmt"
	"os"

	"github.com/prometheus/prometheus/model/relabel"
	"gopkg.in/yaml.v2"
)

// Application configuration
type Config struct {
	Addr           string
	PromConfigPath string
	LogLevel       string
	Env            string
}

func NewConfig(addr, promConfigPath, logLevel, env string) *Config {
	c := &Config{
		Addr:           addr,
		PromConfigPath: promConfigPath,
		LogLevel:       logLevel,
		Env:            env,
	}
	return c
}

// Prometheus relabelling configuration
// Uses same structure as regular prometheus config.
type PromConfig struct {
	ScrapeConfigs []*scrapeConfig `yaml:"scrape_configs"`
}

// scrapeConfig defines a proxied scrape job.
type scrapeConfig struct {
	// The job name to which the job label is set by default.
	JobName string `yaml:"job_name"`
	// The HTTP resource path on which to fetch metrics from targets.
	MetricsPath string `yaml:"metrics_path,omitempty"`
	// The URL scheme with which to fetch metrics from targets.
	Scheme string `yaml:"scheme,omitempty"`
	// The hostnames of scrape targets
	StaticConfigs []*staticConfig `yaml:"static_configs"`
	// List of target relabel configurations.
	RelabelConfigs []*relabel.Config `yaml:"relabel_configs,omitempty"`
	// List of metric relabel configurations.
	MetricRelabelConfigs []*relabel.Config `yaml:"metric_relabel_configs,omitempty"`
}

type staticConfig struct {
	Targets []string `yaml:"targets"`
}

func LoadPromConfig(path string) (*PromConfig, error) {

	c := &PromConfig{}

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
