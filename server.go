package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/cgi"
	"time"
)

type GitCGIServer struct {
	ProjectRoot     string
	ExportAll       bool
	URLPrefix       string
	Addr            string
	ShutdownTimeout time.Duration
	MustClose       bool
	httpServer      *http.Server
}

func (s *GitCGIServer) Serve() error {
	if s.URLPrefix == "" {
		s.URLPrefix = "/"
	}
	mux := http.NewServeMux()
	mux.HandleFunc(s.URLPrefix, s.gitBackend)

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
	log.Println(r.URL)
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

	var stdErr bytes.Buffer
	handler := &cgi.Handler{
		Path:       cgiBin,
		Env:        env,
		InheritEnv: inheritEnv,
		Stderr:     &stdErr,
	}
	handler.ServeHTTP(w, r)

	if stdErr.Len() > 0 {
		log.Println("[backend]", stdErr.String())
	}
}
