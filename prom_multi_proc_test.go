package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
	"testing"
)

var testMetrics string = `[
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

func SetTestLogger() {
	var out bytes.Buffer
	logger = log.New(&out, "", log.LstdFlags)
}

func TestParseMetrics(t *testing.T) {
	SetTestLogger()

	// get metrics handlers
	r := strings.NewReader(testMetrics)
	metrics, err := ParseMetrics(r)
	if err != nil {
		t.Fatal(err)
	}

	// get raw metric specs
	r.Seek(0, 0)
	jsonBlob, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	var specs []MetricSpec
	err = json.Unmarshal(jsonBlob, &specs)
	if err != nil {
		t.Fatal(err)
	}

	if len(metrics) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(specs))
	}

	for name, handler := range metrics {
		var labelValues []string
		if strings.Contains(name, "_vec") {
			labelValues = []string{"a", "b", "c"}
		}
		var methods []string
		for _, spec := range specs {
			if spec.Name == name {
				switch spec.Type {
				default:
					t.Fatal("Invalid metric handler type: %+v", handler)
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
			t.Fatal("No methods found for handler: %+v", handler)
		}
		for _, method := range methods {
			m := Metric{
				Name:        name,
				Method:      method,
				Value:       0.1,
				LabelValues: labelValues,
			}
			handler.Handle(&m)
		}
	}
}
