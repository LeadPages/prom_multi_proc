package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// build flags
var (
	Version   string = "unset"
	BuildTime string = "unset"
	BuildUser string = "unset"
	BuildHash string = "unset"
)

// cli flags
var (
	socketFlag  = flag.String("socket", "/tmp/prom_multi_proc.sock", "Path to unix socket to listen on for incoming metrics")
	metricsFlag = flag.String("metrics", "", "Path to json file which contains metric definitions")
	addrFlag    = flag.String("addr", "0.0.0.0:9299", "Address to listen on for exposing prometheus metrics")
	pathFlag    = flag.String("path", "/metrics", "Path to use for exposing prometheus metrics")
	logFlag     = flag.String("log", "", "Path to log file, will write to STDOUT if empty")
	versionFlag = flag.Bool("v", false, "Print version information and exit")
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("prom_multi_proc %s %s %s %s\n", Version, BuildTime, BuildUser, BuildHash)
		os.Exit(0)
	}

	if *logFlag == "" {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		logFile, err := os.OpenFile(*logFlag, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error opening log file (%s): %s\n", *logFlag, err)
			os.Exit(1)
		}
		logger = log.New(logFile, "", log.LstdFlags)
	}

	metricCh := make(chan *Metric)

	ln, err := net.Listen("unix", *socketFlag)
	if err != nil {
		logger.Fatal(err)
	}

	metricsFile, err := os.OpenFile(*metricsFlag, os.O_RDONLY, 0644)
	if err != nil {
		logger.Fatal(err)
	}

	metrics, err := ParseMetrics(metricsFile)
	if err != nil {
		logger.Fatal(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigc
		ln.Close()
		logger.Println("Goodbye!")
		os.Exit(0)
	}()

	go DataProcessor(metrics, metricCh)
	go DataReader(ln, metricCh)

	promHandler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog: logger,
	})
	http.Handle(*pathFlag, promHandler)
	http.ListenAndServe(*addrFlag, nil)
}
