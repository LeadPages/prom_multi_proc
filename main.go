package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// cli flags
var (
	socketFlag   = flag.String("socket", "/tmp/prom_multi_proc.sock", "Path to unix socket to listen on for incoming metrics")
	handlersFlag = flag.String("handlers", "", "Path to json file which contains metric definitions")
	addrFlag     = flag.String("addr", "0.0.0.0:9299", "Address to listen on for exposing prometheus metrics")
	pathFlag     = flag.String("path", "/metrics", "Path to use for exposing prometheus metrics")
)

func main() {
	flag.Parse()

	metricCh := make(chan *Metric)

	ln, err := net.Listen("unix", *socketFlag)
	if err != nil {
		log.Fatal(err)
	}

	handlers, err := ParseHandlers(*handlersFlag)
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
	http.Handle(*pathFlag, promhttp.Handler())
	http.ListenAndServe(*addrFlag, nil)
}
