package handlers

import "net/http"

type limitHandler struct {
	sem     chan struct{}
	handler http.Handler
}

func (h *limitHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	select {
	case <-h.sem:
		h.handler.ServeHTTP(w, req)
		h.sem <- struct{}{}
	default:
		http.Error(w, "503 too busy", http.StatusServiceUnavailable)
	}
}

func NewLimitHandler(maxConns int, handler http.Handler) http.Handler {
	h := &limitHandler{
		sem:     make(chan struct{}, maxConns),
		handler: handler,
	}
	for i := 0; i < maxConns; i++ {
		h.sem <- struct{}{}
	}
	return h
}
