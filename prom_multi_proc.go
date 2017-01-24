package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricRe = regexp.MustCompile(`^[a-z]+\[[0-9a-z_]+\]$`)
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
	MetricSpec() *MetricSpec
	Handle(*Metric)
}

type CounterHandler struct {
	Spec    MetricSpec
	Counter prometheus.Counter
}

func (h *CounterHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		log.Printf("Invalid counter method %s for metric %s\n", m.Method, m.Name)
	case "inc":
		h.Counter.Inc()
	case "add":
		h.Counter.Add(m.Value)
	}
}

func (h *CounterHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

type CounterVecHandler struct {
	Spec       MetricSpec
	CounterVec *prometheus.CounterVec
}

func (h *CounterVecHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		log.Printf("Invalid counter method %s for metric %s\n", m.Method, m.Name)
	case "inc":
		h.CounterVec.WithLabelValues(m.LabelValues...).Inc()
	case "add":
		h.CounterVec.WithLabelValues(m.LabelValues...).Add(m.Value)
	}
}

func (h *CounterVecHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

type GaugeHandler struct {
	Spec  MetricSpec
	Gauge prometheus.Gauge
}

func (h *GaugeHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		log.Printf("Invalid gauge method %s for metric %s\n", m.Method, m.Name)
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

func (h *GaugeHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

type GaugeVecHandler struct {
	Spec     MetricSpec
	GaugeVec *prometheus.GaugeVec
}

func (h *GaugeVecHandler) Handle(m *Metric) {
	switch m.Method {
	default:
		log.Printf("Invalid gauge vec method %s for metric %s\n", m.Method, m.Name)
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

func (h *GaugeVecHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

type HistogramHandler struct {
	Spec      MetricSpec
	Histogram prometheus.Histogram
}

func (h *HistogramHandler) Handle(m *Metric) {
	h.Histogram.Observe(m.Value)
}

func (h *HistogramHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

type HistogramVecHandler struct {
	Spec         MetricSpec
	HistogramVec *prometheus.HistogramVec
}

func (h *HistogramVecHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

func (h *HistogramVecHandler) Handle(m *Metric) {
	h.HistogramVec.WithLabelValues(m.LabelValues...).Observe(m.Value)
}

type SummaryHandler struct {
	Spec    MetricSpec
	Summary prometheus.Summary
}

func (h *SummaryHandler) Handle(m *Metric) {
	h.Summary.Observe(m.Value)
}

func (h *SummaryHandler) MetricSpec() *MetricSpec {
	return &h.Spec
}

type SummaryVecHandler struct {
	Spec       MetricSpec
	SummaryVec *prometheus.SummaryVec
}

func (h *SummaryVecHandler) Handle(m *Metric) {
	h.SummaryVec.WithLabelValues(m.LabelValues...).Observe(m.Value)
}

func (h *SummaryVecHandler) MetricSpec() *MetricSpec {
	return &h.Spec
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

func ValidateBuckets(buckets []float64) error {
	if len(buckets) == 0 {
		return errors.New("Buckets must not be empty")
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

func ParseHandlers(file string) (map[string]MetricHandler, error) {
	result := make(map[string]MetricHandler)

	jsonBlob, err := ioutil.ReadFile(file)
	if err != nil {
		return result, err
	}

	var specs []MetricSpec
	err = json.Unmarshal(jsonBlob, &specs)
	if err != nil {
		return result, err
	}

	for _, spec := range specs {
		err = ValidateMetric(spec.Name)
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
				h = &CounterHandler{spec, p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewCounterVec(opts, spec.Labels)
				h = &CounterVecHandler{spec, p}
				c = p
			}
		case "gauge":
			opts := prometheus.GaugeOpts{
				Name: spec.Name,
				Help: spec.Help,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewGauge(opts)
				h = &GaugeHandler{spec, p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewGaugeVec(opts, spec.Labels)
				h = &GaugeVecHandler{spec, p}
				c = p
			}
		case "histogram":
			err = ValidateBuckets(spec.Buckets)
			if err != nil {
				return result, err
			}
			opts := prometheus.HistogramOpts{
				Name:    spec.Name,
				Help:    spec.Help,
				Buckets: spec.Buckets,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewHistogram(opts)
				h = &HistogramHandler{spec, p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewHistogramVec(opts, spec.Labels)
				h = &HistogramVecHandler{spec, p}
				c = p
			}
		case "summary":
			objs, err := ValidateObjectives(spec.Objectives)
			if err != nil {
				return result, err
			}
			opts := prometheus.SummaryOpts{
				Name:       spec.Name,
				Help:       spec.Help,
				Objectives: objs,
			}
			if len(spec.Labels) == 0 {
				p := prometheus.NewSummary(opts)
				h = &SummaryHandler{spec, p}
				c = p
			} else {
				err = ValidateLabels(spec.Labels)
				if err != nil {
					return result, err
				}
				p := prometheus.NewSummaryVec(opts, spec.Labels)
				h = &SummaryVecHandler{spec, p}
				c = p
			}
		}

		err = prometheus.Register(c)
		if err != nil {
			return result, err
		}
		result[spec.Name] = h
		log.Printf("Registered %s %s", spec.Type, spec.Name)
	}

	return result, nil
}

func DataReader(ln net.Listener, metricCh chan<- *Metric) {
	for {
		// accept a connection
		c, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		// we create a decoder that reads directly from the connection
		d := json.NewDecoder(c)

		var metric Metric

		err = d.Decode(&metric)
		if err != nil {
			log.Println(err)
			continue
		}

		metricCh <- &metric
		c.Close()
	}
}

func DataProcessor(handlers map[string]MetricHandler, metricCh <-chan *Metric) {
	for {
		metric := <-metricCh
		handler, ok := handlers[metric.Name]
		if !ok {
			log.Printf("Metric %s not found\n", metric.Name)
			continue
		}
		handler.Handle(metric)
	}
}
