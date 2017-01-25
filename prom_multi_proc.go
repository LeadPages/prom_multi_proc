package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricRe       = regexp.MustCompile(`^[a-z]+\[[0-9a-z_]+\]$`)
	defaultBuckets = []float64{
		0.005,
		0.01,
		0.025,
		0.05,
		0.1,
		0.25,
		0.5,
		1.0,
		2.5,
		5.0,
		10.0,
	}
	defaultObjectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	logCloser         io.WriteCloser
	logger            *log.Logger
)

type MetricSpec struct {
	Type       string             `json:"type"`
	Name       string             `json:"name"`
	Help       string             `json:"help"`
	Labels     []string           `json:"labels"`
	Buckets    []float64          `json:"buckets"`
	Objectives map[string]float64 `json:"objectives"`
}

type Metric struct {
	Name        string   `json:"name"`
	LabelValues []string `json:"label_values"`
	Method      string   `json:"method"`
	Value       float64  `json:"value"`
}

type MetricHandler interface {
	Handle(*Metric)
	Collector() prometheus.Collector
}

type CounterHandler struct {
	Counter prometheus.Counter
}

func (h *CounterHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		logger.Printf("Invalid counter method %s for metric %s\n", m.Method, m.Name)
	case "inc":
		h.Counter.Inc()
	case "add":
		h.Counter.Add(m.Value)
	}
}

func (h *CounterHandler) Collector() prometheus.Collector {
	return h.Counter
}

type CounterVecHandler struct {
	CounterVec *prometheus.CounterVec
}

func (h *CounterVecHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		logger.Printf("Invalid counter method %s for metric %s\n", m.Method, m.Name)
	case "inc":
		h.CounterVec.WithLabelValues(m.LabelValues...).Inc()
	case "add":
		h.CounterVec.WithLabelValues(m.LabelValues...).Add(m.Value)
	}
}

func (h *CounterVecHandler) Collector() prometheus.Collector {
	return h.CounterVec
}

type GaugeHandler struct {
	Gauge prometheus.Gauge
}

func (h *GaugeHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		logger.Printf("Invalid gauge method %s for metric %s\n", m.Method, m.Name)
	case "set":
		h.Gauge.Set(m.Value)
	case "inc":
		h.Gauge.Inc()
	case "dec":
		h.Gauge.Dec()
	case "add":
		h.Gauge.Add(m.Value)
	case "sub":
		h.Gauge.Sub(m.Value)
	case "set_to_current_time":
		h.Gauge.SetToCurrentTime()
	}
}

func (h *GaugeHandler) Collector() prometheus.Collector {
	return h.Gauge
}

type GaugeVecHandler struct {
	GaugeVec *prometheus.GaugeVec
}

func (h *GaugeVecHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		logger.Printf("Invalid gauge vec method %s for metric %s\n", m.Method, m.Name)
	case "set":
		h.GaugeVec.WithLabelValues(m.LabelValues...).Set(m.Value)
	case "inc":
		h.GaugeVec.WithLabelValues(m.LabelValues...).Inc()
	case "dec":
		h.GaugeVec.WithLabelValues(m.LabelValues...).Dec()
	case "add":
		h.GaugeVec.WithLabelValues(m.LabelValues...).Add(m.Value)
	case "sub":
		h.GaugeVec.WithLabelValues(m.LabelValues...).Sub(m.Value)
	case "set_to_current_time":
		h.GaugeVec.WithLabelValues(m.LabelValues...).SetToCurrentTime()
	}
}

func (h *GaugeVecHandler) Collector() prometheus.Collector {
	return h.GaugeVec
}

type HistogramHandler struct {
	Histogram prometheus.Histogram
}

func (h *HistogramHandler) Handle(m *Metric) {
	h.Histogram.Observe(m.Value)
}

func (h *HistogramHandler) Collector() prometheus.Collector {
	return h.Histogram
}

type HistogramVecHandler struct {
	HistogramVec *prometheus.HistogramVec
}

func (h *HistogramVecHandler) Handle(m *Metric) {
	h.HistogramVec.WithLabelValues(m.LabelValues...).Observe(m.Value)
}

func (h *HistogramVecHandler) Collector() prometheus.Collector {
	return h.HistogramVec
}

type SummaryHandler struct {
	Summary prometheus.Summary
}

func (h *SummaryHandler) Handle(m *Metric) {
	h.Summary.Observe(m.Value)
}

func (h *SummaryHandler) Collector() prometheus.Collector {
	return h.Summary
}

type SummaryVecHandler struct {
	SummaryVec *prometheus.SummaryVec
}

func (h *SummaryVecHandler) Handle(m *Metric) {
	h.SummaryVec.WithLabelValues(m.LabelValues...).Observe(m.Value)
}

func (h *SummaryVecHandler) Collector() prometheus.Collector {
	return h.SummaryVec
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}

func SetLogger(file string) error {
	if logCloser != nil {
		logCloser.Close()
	}
	var err error
	if file == "" {
		var b bytes.Buffer
		logCloser = nopCloser{&b}
		logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		logCloser, err = os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("Error opening log file (%s): %s\n", file, err)
		}
		logger = log.New(logCloser, "", log.LstdFlags)
	}
	return nil
}

func ValidateMetric(name string) error {
	if !metricRe.MatchString(name) {
		fmt.Errorf("Metric name '%s' is not valid", name)
	}

	return nil
}

func ValidateLabels(labels []string) error {
	n := len(labels)

	for i := 0; i < n-1; i++ {
		err := ValidateMetric(labels[i])
		if err != nil {
			return err
		}

		for j := i + 1; j < n; j++ {
			if labels[i] == labels[j] {
				return fmt.Errorf("Duplicate label found: %s", labels[i])
			}
		}
	}

	return nil
}

func ValidateObjectives(objectives map[string]float64) (map[float64]float64, error) {
	result := make(map[float64]float64)

	for key, value := range objectives {
		f, err := strconv.ParseFloat(key, 64)
		if err != nil {
			return result, err
		}
		result[f] = value
	}

	return result, nil
}

func LoadMetrics(file string) ([]MetricSpec, map[string]MetricHandler, error) {
	var (
		specs    []MetricSpec
		handlers map[string]MetricHandler
		err      error
	)

	metricsFile, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		return specs, handlers, err
	}
	defer metricsFile.Close()

	specs, err = ReadMetrics(metricsFile)
	if err != nil {
		return specs, handlers, err
	}

	handlers, err = ParseMetricSpecs(specs)
	if err != nil {
		return specs, handlers, err
	}

	return specs, handlers, nil
}

func ReadMetrics(r io.Reader) ([]MetricSpec, error) {
	var result []MetricSpec

	jsonBlob, err := ioutil.ReadAll(r)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(jsonBlob, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func ParseMetricSpecs(specs []MetricSpec) (map[string]MetricHandler, error) {
	result := make(map[string]MetricHandler)

	for _, spec := range specs {
		err := ValidateMetric(spec.Name)
		if err != nil {
			return result, err
		}

		if _, ok := result[spec.Name]; ok {
			return result, fmt.Errorf("Metric with name %s already exists\n", spec.Name)
		}

		var (
			c prometheus.Collector
			h MetricHandler
		)
		switch spec.Type {
		default:
			return result, fmt.Errorf("Unknown metric type (%s) for metric %s\n", spec.Type, spec.Name)
		case "counter":
			opts := prometheus.CounterOpts{
				Name: spec.Name,
				Help: spec.Help,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewCounter(opts)
				h = &CounterHandler{p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewCounterVec(opts, spec.Labels)
				h = &CounterVecHandler{p}
				c = p
			}
		case "gauge":
			opts := prometheus.GaugeOpts{
				Name: spec.Name,
				Help: spec.Help,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewGauge(opts)
				h = &GaugeHandler{p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewGaugeVec(opts, spec.Labels)
				h = &GaugeVecHandler{p}
				c = p
			}
		case "histogram":
			var buckets []float64
			if len(spec.Buckets) > 0 {
				buckets = spec.Buckets
			} else {
				buckets = defaultBuckets
			}
			opts := prometheus.HistogramOpts{
				Name:    spec.Name,
				Help:    spec.Help,
				Buckets: buckets,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewHistogram(opts)
				h = &HistogramHandler{p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewHistogramVec(opts, spec.Labels)
				h = &HistogramVecHandler{p}
				c = p
			}
		case "summary":
			var objectives map[float64]float64
			if len(spec.Objectives) > 0 {
				objectives, err = ValidateObjectives(spec.Objectives)
				if err != nil {
					return result, err
				}
			} else {
				objectives = defaultObjectives
			}
			opts := prometheus.SummaryOpts{
				Name:       spec.Name,
				Help:       spec.Help,
				Objectives: objectives,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewSummary(opts)
				h = &SummaryHandler{p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewSummaryVec(opts, spec.Labels)
				h = &SummaryVecHandler{p}
				c = p
			}
		}

		err = prometheus.Register(c)
		if err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
				// this collector is already registered
				// we log the message because this could happen on USR1 reload
				// clients should know they can't modify existing metrics w/o killing & restarting the process
				logger.Printf("Attempt to re-register metric '%s', not updating", spec.Name)
			} else {
				return result, err
			}
		} else {
			result[spec.Name] = h
			logger.Printf("Registered %s %s", spec.Type, spec.Name)
		}
	}

	return result, nil
}

func Unregister(handlers map[string]MetricHandler, names []string) {
	for _, name := range names {
		if handler, ok := handlers[name]; ok {
			result := prometheus.Unregister(handler.Collector())
			if result {
				logger.Printf("Unregistered %s", name)
			} else {
				logger.Printf("Failed to unregistered %s", name)
			}
		}
	}
}

func DataReader(ln net.Listener, metricCh chan<- *Metric) {
	for {
		// accept a connection
		c, err := ln.Accept()
		if err != nil {
			logger.Println(err)
			continue
		}

		// we create a decoder that reads directly from the connection
		d := json.NewDecoder(c)

		var metric Metric

		err = d.Decode(&metric)
		if err != nil {
			logger.Println(err)
			continue
		}

		metricCh <- &metric
		c.Close()
	}
}

func DataProcessor(handlers map[string]MetricHandler, metricCh <-chan *Metric, doneCh <-chan bool) {
	for {
		select {
		case metric := <-metricCh:
			handler, ok := handlers[metric.Name]
			if !ok {
				logger.Printf("Metric %s not found\n", metric.Name)
				continue
			}
			handler.Handle(metric)
		case <-doneCh:
			break
		}
	}
}
