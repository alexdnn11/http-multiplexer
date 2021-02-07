package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
)

const (
	unsupportedUrlsNumber = "unsupported number of urls"
)

func main() {

	var (
		maxWorkers = flag.Int("max_workers", 5, "The number of workers to start")
		maxClients = flag.Int("max_clients", 100, "The number of clients to handle request concurrently")
		port       = flag.String("port", "8080", "The server port")
	)
	flag.Parse()

	http.HandleFunc("/", limitNumClients(handler, *maxClients))

	log.Printf("Going to listen on port %s\n", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {

	const maxUrls = 20
	var (
		urls []string
		err  error
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

	if len(urls) > maxUrls {
		http.Error(w, unsupportedUrlsNumber, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func limitNumClients(f http.HandlerFunc, maxClients int) http.HandlerFunc {
	sema := make(chan struct{}, maxClients)

	return func(w http.ResponseWriter, req *http.Request) {
		sema <- struct{}{}
		defer func() { <-sema }()
		f(w, req)
	}
}
