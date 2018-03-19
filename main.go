package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// TOOD: be configurable
const (
	SERVER_ADDR      = ":8080"
	SHUTDOWN_TIMEOUT = time.Second * 5
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	server := &GitCGIServer{
		ProjectRoot:     cwd,
		Addr:            SERVER_ADDR,
		ShutdownTimeout: SHUTDOWN_TIMEOUT,
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
			log.Println("Http server error:", err)
		}

	case sig := <-sigCh:
		log.Printf("Signal %s received\n", sig)
		if err := server.Shutdown(); err != nil {
			log.Println("Failed to shutdown HTTP server:", err)
		}
		log.Println("HTTP server shutdown")
	}
}
