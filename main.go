package main

import (
	"flag"
	"fmt"
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

	// setup logger, this may be reloaded later with HUP signal
	err := SetLogger(*logFlag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// setup metrics and done channels
	metricCh := make(chan *Metric)
	doneCh := make(chan bool)

	// begin listening on socket
	ln, err := net.Listen("unix", *socketFlag)
	if err != nil {
		logger.Fatal(err)
	}

	err = os.Chmod(*socketFlag, 0777)
	if err != nil {
		logger.Fatal(err)
	}

	// listen for signals which make us quit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go func() {
		<-sigc
		ln.Close()
		logger.Println("Goodbye!")
		os.Exit(0)
	}()

	// load initial metrics from file, this may be reloaded later with USR1 signal
	specs, handlers, err := LoadMetrics(*metricsFlag)
	if err != nil {
		logger.Printf("Error loading metrics definitions: %s", err)
		// start with empty specs and handlers
		specs = []MetricSpec{}
		handlers = map[string]MetricHandler{}
	}

	// begin processing initial metrics definitions
	go DataProcessor(handlers, metricCh, doneCh)

	// listen for USR1 signal which makes us reload our metrics definitions
	sigu := make(chan os.Signal, 1)
	signal.Notify(sigu, syscall.SIGUSR1)
	go func() {
		for {
			<-sigu
			logger.Println("Re-loading configuration...")

			// note names of original metrics
			originalNames := metricNames(specs)

			// reload metrics definitions file
			newSpecs, newHandlers, err := LoadMetrics(*metricsFlag)
			if err != nil {
				logger.Printf("Error re-loading configuration: %s", err)
				continue
			}

			// stop the old data processor
			doneCh <- true

			// add newly registered specs and handlers
			for name, handler := range newHandlers {
				handlers[name] = handler
			}

			// get names of metrics no longer present and unregister them
			newNames := metricNames(newSpecs)
			unreg := sliceSubStr(originalNames, newNames)
			Unregister(handlers, unreg)

			// delete unregistered handlers
			for _, name := range unreg {
				delete(handlers, name)
			}

			specs = newSpecs

			// begin processing incoming metrics again
			go DataProcessor(handlers, metricCh, doneCh)
		}
	}()

	// listen for HUP signal which makes us reopen our log file descriptors
	sigh := make(chan os.Signal, 1)
	signal.Notify(sigh, syscall.SIGHUP)
	go func() {
		for {
			<-sigh
			logger.Println("Re-opening logs...")
			err := SetLogger(*logFlag)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}()

	// begin reading off socket and sending results into metrics channel
	go DataReader(ln, metricCh)

	// setup prometheus http handlers and begin listening
	promHandler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog: logger,
	})
	http.Handle(*pathFlag, promHandler)
	http.ListenAndServe(*addrFlag, nil)
}
