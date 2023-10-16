# prom-relabel-proxy

Simple proxy to enable relabelling and modification of prometheus metrics

## Description

:warning: you probably don't need this

Prometheus offers powerful relabelling rules when scraping targets.
If you control the prometheus scrape config, it is **highly** recommended
to apply relabel rules there. This proxy is only useful if you cannot apply
relabel rules in your prometheus service.

This proxy sits in front of any number of scrape targets. When a scrape request
is sent to the proxy, it will scrape all configured targets, apply any relabel rules
and return the formatted metrics to the original requester.

## Getting Started

### Running

The proxy can be run from a pre-built docker container:

```yaml
version: "3.7"
services:
  prom-relabel-proxy:
    container_name: prom-relabel-proxy
    image: evanofslack/prom-relabel-proxy:latest
    ports:
      - 9091:9091
    restart: unless-stopped
    volumes:
      - ./config:/config
    command: -c /config/relabel.yaml
```

Alternatively, build the executable from source:

```bash
git clone https://github.com/evanofslack/prom-relabel-proxy
cd prom-relabel-proxy/cmd/prom-relabel-proxy
go build ./...
```

### Parameters

Runtime parameters can be configured through flags:

```bash
usage: prom-relabel-proxy [-a listen address] [-c relabel config path] [-e app environment] [-l logLevel] [-m metrics path]
  -a string
        address proxy listens on (default ":9091")
  -c string
        path to prometheus relabel config (default "relabel.yaml")
  -e string
        app environment [debug, prod] (default "prod")
  -l string
        log level [debug, info, warn, error] (default "info")
  -m string
        path to serve metrics from (default "/metrics")

```

### Configuration

The expected configuration file takes the same format as a Prometheus
scrape configuration file. For example, when running prometheus and
the relabel proxy as containers in the same network, we can relabel
prometheus's internal metrics with the following config `relabel.yaml`.

```yaml
scrape_configs:
  - job_name: "prometheus"
    metrics_path: "/metrics"
    scheme: "http"
    static_configs:
      - targets: ["prometheus:9090"]
    relabel_configs:
      - source_labels: [__name__]
        regex: "prometheus_http_requests_total"
        target_label: handler
        replacement: ""
      - source_labels: [__name__]
        regex: "prometheus_http_requests_total"
        target_label: "added_label"
        replacement: "example"
```

The relabel proxy can take any number of `scrape_config` jobs. It will
scrape each one, apply relabel rules, and combine into one output, ready
to be scraped.
