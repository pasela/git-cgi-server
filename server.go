package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"html/template"
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
	GoModules       bool
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

func (s *GitCGIServer) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.ShutdownTimeout)
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
	s.handleRequest(w, &ar.Request, ar.Username)
}

func (s *GitCGIServer) gitNoAuthHandler(w http.ResponseWriter, r *http.Request) {
	s.handleRequest(w, r, "")
}

func (s *GitCGIServer) handleRequest(w http.ResponseWriter, r *http.Request, username string) {
	if s.GoModules && isGoGetRequest(r) {
		s.handleGoGetRequest(w, r)
	} else {
		s.gitBackend(w, r, username)
	}
}

func isGoGetRequest(r *http.Request) bool {
	return r.URL.Query().Has("go-get")
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

const goModulesTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
   	<title>{{.ModulePath}}</title>
    <meta name="go-import" content="{{.ModulePath}} {{.Vcs}} {{.RepoURL}}">
  </head>
  <body>
   	<p><code>go get {{.ModulePath}}</code></p>
  </body>
</html>
`

type goModuleInfo struct {
	Vcs        string
	ModulePath string
	RepoURL    string
}

type gitRepoInfo struct {
	RequestPath string // Request path (e.g. "/URI-PREFIX/foo")
	RequestDir  string // Request directory (e.g. "/PROJECT-ROOT/foo")
	GitPath     string // Git repository path (e.g. "/URI-PREFIX/foo.git")
	GitDir      string // Git repository directory (e.g. "/PROJECT-ROOT/foo.git")
	Exists      bool
}

func (s *GitCGIServer) handleGoGetRequest(w http.ResponseWriter, r *http.Request) {
	if !isGoGetRequest(r) {
		w.WriteHeader(404)
		return
	}

	repo := s.getGitRepoInfo(r.URL.Path)
	if !repo.Exists {
		w.WriteHeader(404)
		return
	}

	modulePath := r.Host + repo.RequestPath
	port := toURLPort(s.Addr)
	var repoURL string
	if r.TLS != nil {
		repoURL = "https://" + r.Host + port + repo.GitPath
	} else {
		repoURL = "http://" + r.Host + port + repo.GitPath
	}
	data := goModuleInfo{
		Vcs:        "git",
		ModulePath: modulePath,
		RepoURL:    repoURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	t := template.Must(template.New("go-modules").Parse(goModulesTemplate))
	err := t.Execute(w, data)
	if err != nil {
		log.Println(err)
	}
}

func (s *GitCGIServer) getGitRepoInfo(requestPath string) gitRepoInfo {
	requestRepo := stripPrefix(requestPath, s.URIPrefix)
	requestDir := filepath.Join(s.ProjectRoot, requestRepo)

	gitPath := requestPath + ".git"
	gitDir := requestDir + ".git"
	hasRequestDir := isDir(requestDir)
	hasGitDir := isDir(gitDir)

	if !hasGitDir && hasRequestDir {
		gitPath = requestPath
		gitDir = requestDir
	}

	return gitRepoInfo{
		RequestPath: requestPath,
		RequestDir:  requestDir,
		GitPath:     gitPath,
		GitDir:      gitDir,
		Exists:      hasGitDir || hasRequestDir,
	}
}
