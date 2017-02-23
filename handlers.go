package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func ValidateVecLabels(h MetricHandler, m *Metric) error {
	if len(h.Spec().Labels) == len(m.LabelValues) {
		return nil
	}

	return fmt.Errorf("Invalid labels (%s): need ('%s'), got ('%s')",
		m.Name, strings.Join(h.Spec().Labels, "','"), strings.Join(m.LabelValues, "','"))
}

type MetricHandler interface {
	Spec() *MetricSpec
	Handle(*Metric) error
	Collector() prometheus.Collector
}

type CounterHandler struct {
	spec    *MetricSpec
	Counter prometheus.Counter
}

func (h *CounterHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *CounterHandler) Handle(m *Metric) error {
	switch m.Method {
	default:
		logger.Printf("Invalid counter method %s for metric %s\n", m.Method, m.Name)
	case "inc":
		h.Counter.Inc()
	case "add":
		h.Counter.Add(m.Value)
	}

	return nil
}

func (h *CounterHandler) Collector() prometheus.Collector {
	return h.Counter
}

type CounterVecHandler struct {
	spec       *MetricSpec
	CounterVec *prometheus.CounterVec
}

func (h *CounterVecHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *CounterVecHandler) Handle(m *Metric) error {
	if err := ValidateVecLabels(h, m); err != nil {
		return err
	}

	switch m.Method {
	default:
		logger.Printf("Invalid counter method %s for metric %s\n", m.Method, m.Name)
	case "inc":
		h.CounterVec.WithLabelValues(m.LabelValues...).Inc()
	case "add":
		h.CounterVec.WithLabelValues(m.LabelValues...).Add(m.Value)
	}

	return nil
}

func (h *CounterVecHandler) Collector() prometheus.Collector {
	return h.CounterVec
}

type GaugeHandler struct {
	spec  *MetricSpec
	Gauge prometheus.Gauge
}

func (h *GaugeHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *GaugeHandler) Handle(m *Metric) error {
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

	return nil
}

func (h *GaugeHandler) Collector() prometheus.Collector {
	return h.Gauge
}

type GaugeVecHandler struct {
	spec     *MetricSpec
	GaugeVec *prometheus.GaugeVec
}

func (h *GaugeVecHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *GaugeVecHandler) Handle(m *Metric) error {
	if err := ValidateVecLabels(h, m); err != nil {
		return err
	}

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

	return nil
}

func (h *GaugeVecHandler) Collector() prometheus.Collector {
	return h.GaugeVec
}

type HistogramHandler struct {
	spec      *MetricSpec
	Histogram prometheus.Histogram
}

func (h *HistogramHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *HistogramHandler) Handle(m *Metric) error {
	h.Histogram.Observe(m.Value)
	return nil
}

func (h *HistogramHandler) Collector() prometheus.Collector {
	return h.Histogram
}

type HistogramVecHandler struct {
	spec         *MetricSpec
	HistogramVec *prometheus.HistogramVec
}

func (h *HistogramVecHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *HistogramVecHandler) Handle(m *Metric) error {
	if err := ValidateVecLabels(h, m); err != nil {
		return err
	}

	h.HistogramVec.WithLabelValues(m.LabelValues...).Observe(m.Value)
	return nil
}

func (h *HistogramVecHandler) Collector() prometheus.Collector {
	return h.HistogramVec
}

type SummaryHandler struct {
	spec    *MetricSpec
	Summary prometheus.Summary
}

func (h *SummaryHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *SummaryHandler) Handle(m *Metric) error {
	h.Summary.Observe(m.Value)
	return nil
}

func (h *SummaryHandler) Collector() prometheus.Collector {
	return h.Summary
}

type SummaryVecHandler struct {
	spec       *MetricSpec
	SummaryVec *prometheus.SummaryVec
}

func (h *SummaryVecHandler) Spec() *MetricSpec {
	return h.spec
}

func (h *SummaryVecHandler) Handle(m *Metric) error {
	if err := ValidateVecLabels(h, m); err != nil {
		return err
	}

	h.SummaryVec.WithLabelValues(m.LabelValues...).Observe(m.Value)
	return nil
}

func (h *SummaryVecHandler) Collector() prometheus.Collector {
	return h.SummaryVec
}
