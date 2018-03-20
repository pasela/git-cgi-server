package main

import (
	"context"
	"log"
	"net/http"
	"net/http/cgi"
	"time"
)

const (
	DefaultAddr = ":8080"
)

type GitCGIServer struct {
	ProjectRoot     string
	ExportAll       bool
	Addr            string
	ShutdownTimeout time.Duration
	MustClose       bool
	httpServer      *http.Server
}

func (s *GitCGIServer) Serve() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", s.hello)
	mux.HandleFunc("/git", s.gitBackend)

	if s.Addr == "" {
		s.Addr = DefaultAddr
	}

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

func (s *GitCGIServer) gitBackend(w http.ResponseWriter, r *http.Request) {
	cgiBin, err := findBackendCGI()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	env := []string{
		"GIT_PROJECT_ROOT=" + s.ProjectRoot,
	}
	if s.ExportAll {
		env = append(env, "GIT_HTTP_EXPORT_ALL=")
	}

	inheritEnv := []string{
		"REMOTE_USER",
	}

	handler := &cgi.Handler{
		Path:       cgiBin,
		Env:        env,
		InheritEnv: inheritEnv,
	}
	handler.ServeHTTP(w, r)
}

func (s *GitCGIServer) hello(w http.ResponseWriter, r *http.Request) {
	log.Println("hello called")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("hello\n"))
}
