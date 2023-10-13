package main

import (
	"fmt"
	"os"

	"github.com/prometheus/prometheus/model/relabel"
	"gopkg.in/yaml.v2"
)

type staticConfig struct {
	Targets []string `yaml:"targets"`
}

// scrapeConfig defines a proxied scrape job.
// Uses same structure as regular prometheus config.
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

type promConfig struct {
	ScrapeConfigs []*scrapeConfig `yaml:"scrape_configs"`
}

func loadPromConfig(path string) (*promConfig, error) {

	c := &promConfig{}

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
