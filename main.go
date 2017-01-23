package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	socketPath := "/tmp/prom_multi_proc.sock"
	handlersFile := "/tmp/handlers.json"
	addr := "0.0.0.0:9299"

	metricCh := make(chan *Metric)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}

	handlers, err := ParseHandlers(handlersFile)
	if err != nil {
		log.Fatal(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigc
		ln.Close()
		log.Println("Goodbye!")
		os.Exit(0)
	}()

	go DataProcessor(handlers, metricCh)
	go DataReader(ln, metricCh)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(addr, nil)
}
