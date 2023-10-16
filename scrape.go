package proxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type Scraper struct {
	logger *slog.Logger
	client *http.Client
}

func NewScraper(logger *slog.Logger) *Scraper {
	s := &Scraper{
		logger: logger,
		client: &http.Client{},
	}

	return s
}

func (s *Scraper) scrape(url string, req *http.Request) ([]byte, error) {

	s.logger.Debug(fmt.Sprintf("new http request for %s", url))

	var buf bytes.Buffer

	// Form a new request to the scrape target
	proxyReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return buf.Bytes(), err
	}

	// Copy over headers
	s.copyHeader(proxyReq.Header, req.Header)

	// Set forwarded for header
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		s.appendHostToXForwardHeader(proxyReq.Header, clientIP)
	}

	res, err := s.client.Do(proxyReq)
	if err != nil {
		return buf.Bytes(), err
	}
	defer res.Body.Close()

	// Check if need to unzip
	var reader io.ReadCloser
	if res.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(res.Body)
		defer reader.Close()
		if err != nil {
			return buf.Bytes(), err
		}
	} else {
		reader = res.Body
	}

	if _, err := io.Copy(&buf, reader); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil

}

func (s *Scraper) copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			s.logger.Debug(fmt.Sprintf("copy header %s | %s", k, v))
			dst.Add(k, v)
		}
	}
}

func (s *Scraper) appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	k := "X-Forwarded-For"
	v := host
	header.Set(k, v)
	s.logger.Debug(fmt.Sprintf("set header %s | %s", k, v))
}
