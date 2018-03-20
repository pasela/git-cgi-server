package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	shutdownTimeout = time.Second * 5
)

func main() {
	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Fatalln(err)
	}

	server := &GitCGIServer{
		ProjectRoot:     projectRoot,
		ShutdownTimeout: shutdownTimeout,
	}

	errCh := make(chan error)
	go func() {
		if err := server.Serve(); err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	log.Printf("Starting HTTP server on %s (PID=%d)\n", server.Addr, os.Getpid())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case err, ok := <-errCh:
		if ok {
			log.Println("HTTP server error:", err)
		}

	case sig := <-sigCh:
		log.Printf("Signal %s received\n", sig)
		if err := server.Shutdown(); err != nil {
			log.Println("Failed to shutdown HTTP server:", err)
		}
		log.Println("HTTP server shutdown")
	}
}

func getProjectRoot() (string, error) {
	if len(os.Args) > 1 && os.Args[1] != "" {
		return os.Args[1], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	return cwd, nil
}
