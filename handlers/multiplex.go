package handlers

import (
	"encoding/json"
	"github.com/alexdnn11/http-multiplexer/pool"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	unsupportedUrlsNumber = "unsupported number of urls"
)

type multiplex struct {
	Collector *pool.Collector
	MaxUrls   int
	Timeout   int
}

func NewMultiplexHandler(c *pool.Collector, mu int, t int) http.Handler {
	return &multiplex{Collector: c, MaxUrls: mu, Timeout: t}
}

func (h *multiplex) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var (
		urls   []string
		err    error
		result = make(map[int]map[string]string)
	)

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(urls) > h.MaxUrls {
		http.Error(w, unsupportedUrlsNumber, http.StatusBadRequest)
		return
	}

	outJob := make(chan map[string]string, len(urls))
	errJob := make(chan error)
	doRequest := func(url string, outCh chan map[string]string, errCh chan error) {
		// Create HTTP client with timeout
		client := &http.Client{
			Timeout: time.Duration(h.Timeout) * time.Second,
		}
		// Make request
		response, err := client.Get(url)
		if err != nil {
			errCh <- err
			return
		}

		content, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errCh <- err
			return
		}
		defer response.Body.Close()

		outCh <- map[string]string{url: string(content)}
	}

	go func() {
		for i, in := range urls {
			h.Collector.Work <- pool.Job{
				ID:     i,
				Input:  in,
				Output: outJob,
				Err:    errJob,
				Do:     doRequest,
			}
		}
	}()

	for j := 0; j < len(urls); j++ {
		select {
		case err := <-errJob:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		case out := <-outJob:
			result[j] = out
		}
	}

	jData, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)

	w.WriteHeader(http.StatusOK)
}
