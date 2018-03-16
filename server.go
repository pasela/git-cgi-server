package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

type GitCGIServer struct {
	Addr            string
	ShutdownTimeout time.Duration
	MustClose       bool
	httpServer      *http.Server
}

func (s *GitCGIServer) Serve() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", s.hello)

	s.httpServer = &http.Server{
		Addr:    s.Addr,
		Handler: mux,
	}

	if err := s.httpServer.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			return err
		}
	}

	return nil
}

func (s *GitCGIServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		if s.MustClose {
			s.httpServer.Close()
		}
		return err
	}

	return nil
}

func (s *GitCGIServer) hello(w http.ResponseWriter, r *http.Request) {
	log.Println("hello called")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("hello\n"))
}
