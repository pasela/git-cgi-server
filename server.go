package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/http/cgi"
	"path"
	"path/filepath"
	"time"

	auth "github.com/abbot/go-http-auth"
)

type GitCGIServer struct {
	ProjectRoot     string
	ExportAll       bool
	BackendCGI      string
	URIPrefix       string
	BasicAuthFile   string
	DigestAuthFile  string
	AuthRealm       string
	Addr            string
	CertFile        string
	KeyFile         string
	ShutdownTimeout time.Duration
	MustClose       bool
	httpServer      *http.Server
}

func (s *GitCGIServer) Serve() error {
	if s.BackendCGI == "" {
		cgiBin, err := findBackendCGI()
		if err != nil {
			return err
		}
		s.BackendCGI = cgiBin
	}

	proot, err := filepath.Abs(s.ProjectRoot)
	if err != nil {
		return err
	}
	s.ProjectRoot = proot

	s.URIPrefix = subtreePath(s.URIPrefix)
	mux := http.NewServeMux()
	mux.HandleFunc(s.URIPrefix, s.getHandler())

	if s.CertFile != "" {
		return s.serveTLS(mux)
	}
	return s.serve(mux)
}

func (s *GitCGIServer) serve(mux *http.ServeMux) error {
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

func (s *GitCGIServer) serveTLS(mux *http.ServeMux) error {
	// See: https://github.com/denji/golang-tls
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	s.httpServer = &http.Server{
		Addr:      s.Addr,
		Handler:   mux,
		TLSConfig: cfg,
	}

	if err := s.httpServer.ListenAndServeTLS(s.CertFile, s.KeyFile); err != nil {
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

func (s *GitCGIServer) getHandler() http.HandlerFunc {
	authenticator := s.getAuthenticator()
	if authenticator != nil {
		return authenticator.Wrap(s.gitAuthHandler)
	}
	return s.gitNoAuthHandler
}

func (s *GitCGIServer) getAuthenticator() auth.AuthenticatorInterface {
	if s.DigestAuthFile != "" {
		secrets := auth.HtdigestFileProvider(s.DigestAuthFile)
		return auth.NewDigestAuthenticator(s.AuthRealm, secrets)
	} else if s.BasicAuthFile != "" {
		secrets := auth.HtpasswdFileProvider(s.BasicAuthFile)
		return auth.NewBasicAuthenticator(s.AuthRealm, secrets)
	}
	return nil
}

func (s *GitCGIServer) gitAuthHandler(w http.ResponseWriter, ar *auth.AuthenticatedRequest) {
	s.gitBackend(w, &ar.Request, ar.Username)
}

func (s *GitCGIServer) gitNoAuthHandler(w http.ResponseWriter, r *http.Request) {
	s.gitBackend(w, r, "")
}

func (s *GitCGIServer) gitBackend(w http.ResponseWriter, r *http.Request, username string) {
	env := []string{
		"GIT_PROJECT_ROOT=" + s.ProjectRoot,
	}
	if s.ExportAll {
		env = append(env, "GIT_HTTP_EXPORT_ALL=")
	}

	if username != "" {
		env = append(env, "REMOTE_USER="+username)
	}

	var stdErr bytes.Buffer
	handler := &cgi.Handler{
		Path:   s.BackendCGI,
		Root:   path.Clean(s.URIPrefix),
		Env:    env,
		Stderr: &stdErr,
	}
	handler.ServeHTTP(w, r)

	if stdErr.Len() > 0 {
		log.Println("[backend]", stdErr.String())
	}
}
