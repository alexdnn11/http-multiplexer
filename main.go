package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexdnn11/http-multiplexer/handlers"
	"github.com/alexdnn11/http-multiplexer/pool"
)

const (
	defaultBindAddr = ":8080"

	// defaultMaxConn is the default number of max connections the
	// server will handle. 0 means no limits will be set, so the
	// server will be bound by system resources.
	defaultMaxConn    = 100
	defaultMaxUrls    = 20
	defaultReqTimeout = 1
	defaultMaxWorkers = 4
)

func main() {
	var (
		bindAddr   string
		maxConn    uint64
		maxUrls    uint64
		reqTimeout uint64
		maxWorkers uint64
	)

	flag.StringVar(&bindAddr, "b", defaultBindAddr, "TCP address the server will bind to")
	flag.Uint64Var(&maxWorkers, "w", defaultMaxWorkers, "The number of workers to start")
	flag.Uint64Var(&maxConn, "c", defaultMaxConn, "maximum number of client connections the server will accept, 0 means unlimited")
	flag.Uint64Var(&maxUrls, "u", defaultMaxUrls, "maximum number of urls")
	flag.Uint64Var(&reqTimeout, "t", defaultReqTimeout, "request timeout in sec")
	flag.Parse()

	collector := pool.StartDispatcher(int(maxWorkers)) // start up worker pool

	multiplex := handlers.NewMultiplexHandler(&collector, int(maxUrls), int(reqTimeout))
	limiter := handlers.NewLimitHandler(int(maxConn), multiplex)

	router := http.NewServeMux()
	router.Handle("/", limiter)

	srv := http.Server{
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 10,
		Handler:           router,
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s\n", listener.Addr().String())

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Printf("interrupted, shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	collector.End <- true
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v\n", err)
	}
}
