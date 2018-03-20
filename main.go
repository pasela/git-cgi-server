package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultAddr     = ":8080"
	shutdownTimeout = time.Second * 5
)

type Args struct {
	ProjectRoot string
	ExportAll   bool
	Addr        string
}

func parseArgs() (*Args, error) {
	var args Args

	flag.BoolVar(&args.ExportAll, "export-all", false, "export all repositories")
	flag.StringVar(&args.Addr, "addr", defaultAddr, "server address")
	flag.Parse()

	projectRoot, err := getProjectRoot(flag.Args())
	if err != nil {
		return nil, err
	}
	args.ProjectRoot = projectRoot

	return &args, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
		log.Fatalln(err)
	}

	server := &GitCGIServer{
		ProjectRoot:     args.ProjectRoot,
		ExportAll:       args.ExportAll,
		Addr:            args.Addr,
		ShutdownTimeout: shutdownTimeout,
	}

	errCh := make(chan error)
	go func() {
		if err := server.Serve(); err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	log.Printf("Starting HTTP server on %s (PID=%d)\n", args.Addr, os.Getpid())

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

func getProjectRoot(args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	return cwd, nil
}
