package proxy

import (
	"bytes"
	"io"
	"net/http"
)

type Scraper struct {
	client *http.Client
}

func NewScraper() *Scraper {
	s := &Scraper{
		client: &http.Client{},
	}

	return s
}

func (s *Scraper) scrape(path string) ([]byte, error) {
	var buf bytes.Buffer
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return buf.Bytes(), err
	}

	res, err := s.client.Do(req)
	if err != nil {
		return buf.Bytes(), err
	}
	defer res.Body.Close()

	if _, err := io.Copy(&buf, res.Body); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil

}
