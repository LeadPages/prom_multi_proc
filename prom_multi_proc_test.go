package main

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

var (
	testMetrics string = `[
	{
		"type": "counter",
		"name": "test_counter",
		"help": "A test counter"
	},
	{
		"type": "counter",
		"name": "test_counter_vec",
		"help": "A test counter vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "gauge",
		"name": "test_gauge",
		"help": "A test gauge"
	},
	{
		"type": "gauge",
		"name": "test_gauge_vec",
		"help": "A test gauge vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "histogram",
		"name": "test_histogram",
		"help": "A test histogram"
	},
	{
		"type": "histogram",
		"name": "test_histogram_vec",
		"help": "A test histogram vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "histogram",
		"name": "test_histogram_vec_buckets",
		"help": "A test histogram vector",
		"labels": [
			"one",
			"two",
			"three"
		],
		"buckets": [0.1, 0.5, 0.9]
	},
	{
		"type": "summary",
		"name": "test_summary",
		"help": "A test summary"
	},
	{
		"type": "summary",
		"name": "test_summary_vec",
		"help": "A test summary vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "summary",
		"name": "test_summary_vec_objectives",
		"help": "A test summary vector",
		"labels": [
			"one",
			"two",
			"three"
		],
		"objectives": {
			"0.1": 0.1,
			"0.5": 0.5,
			"0.9": 0.9
		}
	}
]`
	handlers map[string]MetricHandler
	specs    []MetricSpec
	mu       sync.Mutex
)

func SetTestLogger() {
	var out bytes.Buffer
	logger = log.New(&out, "", log.LstdFlags)
}

func GetTestHandlers(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()

	if handlers != nil {
		return
	}

	// get metrics handlers
	r := strings.NewReader(testMetrics)
	var err error

	specs, err = ReadMetrics(r)
	if err != nil {
		t.Fatal(err)
	}

	handlers, err = ParseMetricSpecs(specs)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseMetrics(t *testing.T) {
	SetTestLogger()
	GetTestHandlers(t)

	if len(handlers) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(handlers))
	}

	var err error

	for name, handler := range handlers {
		var labelValues []string
		if strings.Contains(name, "_vec") {
			labelValues = []string{"a", "b", "c"}
		}
		var methods []string
		for _, spec := range specs {
			if spec.Name == name {
				switch spec.Type {
				default:
					t.Fatalf("Invalid metric handler type: %+v", handler)
				case "counter", "counter_vec":
					methods = []string{"inc", "add"}
				case "gauge", "gauge_vec":
					methods = []string{"set", "inc", "dec", "add", "sub", "set_to_current_time"}
				case "histogram", "histogram_vec", "summary", "summary_vec":
					methods = []string{"observe"}
				}
				break
			}
		}
		if len(methods) == 0 {
			t.Fatalf("No methods found for handler: %+v", handler)
		}
		for _, method := range methods {
			m := Metric{
				Name:        name,
				Method:      method,
				Value:       1.0,
				LabelValues: labelValues,
			}
			err = handler.Handle(&m)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestParseMetricsFail(t *testing.T) {
	SetTestLogger()
	GetTestHandlers(t)

	if len(handlers) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(handlers))
	}

	var err error

	for name, handler := range handlers {
		var labelValues []string
		if strings.Contains(name, "_vec") {
			labelValues = []string{"a", "b", "c", "d"}
		}
		var methods []string
		for _, spec := range specs {
			if spec.Name == name {
				switch spec.Type {
				default:
					t.Fatalf("Invalid metric handler type: %+v", handler)
				case "counter", "counter_vec":
					methods = []string{"inc", "add"}
				case "gauge", "gauge_vec":
					methods = []string{"set", "inc", "dec", "add", "sub", "set_to_current_time"}
				case "histogram", "histogram_vec", "summary", "summary_vec":
					methods = []string{"observe"}
				}
				break
			}
		}
		if len(methods) == 0 {
			t.Fatalf("No methods found for handler: %+v", handler)
		}
		for _, method := range methods {
			m := Metric{
				Name:        name,
				Method:      method,
				Value:       1.0,
				LabelValues: labelValues,
			}
			err = handler.Handle(&m)
			if strings.Contains(name, "_vec") {
				// here we expect failure due to label length miss-match
				if err == nil {
					t.Fatal(err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
