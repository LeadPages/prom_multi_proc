package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

func getTestSpecs(t *testing.T, i int) []*MetricSpec {
	specsStr := fmt.Sprintf(`[
	{
		"type": "counter",
		"name": "test_%d_counter",
		"help": "Test %d counter"
	},
	{
		"type": "counter",
		"name": "test_%d_counter_vec",
		"help": "Test %d counter vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "gauge",
		"name": "test_%d_gauge",
		"help": "Test %d gauge"
	},
	{
		"type": "gauge",
		"name": "test_%d_gauge_vec",
		"help": "Test %d gauge vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "histogram",
		"name": "test_%d_histogram",
		"help": "Test %d histogram"
	},
	{
		"type": "histogram",
		"name": "test_%d_histogram_vec",
		"help": "Test %d histogram vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "histogram",
		"name": "test_%d_histogram_vec_buckets",
		"help": "Test %d histogram vector",
		"labels": [
			"one",
			"two",
			"three"
		],
		"buckets": [0.1, 0.5, 0.9]
	},
	{
		"type": "summary",
		"name": "test_%d_summary",
		"help": "Test %d summary"
	},
	{
		"type": "summary",
		"name": "test_%d_summary_vec",
		"help": "Test %d summary vector",
		"labels": [
			"one",
			"two",
			"three"
		]
	},
	{
		"type": "summary",
		"name": "test_%d_summary_vec_objectives",
		"help": "Test %d summary vector",
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
]`, []interface{}{i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i}...)
	specsReader := strings.NewReader(specsStr)
	specs, err := ReadSpecs(specsReader)
	if err != nil {
		t.Fatal(err)
	}

	return specs
}

func SetTestLogger() {
	var out bytes.Buffer
	logger = log.New(&out, "", log.LstdFlags)
}

func TestMetrics1(t *testing.T) {
	SetTestLogger()
	specs := getTestSpecs(t, 1)

	if len(specs) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(specs))
	}

	registry := NewRegistry()
	for _, spec := range specs {
		if err := registry.Register(spec); err != nil {
			t.Fatal(err)
		}
	}

	for _, spec := range specs {
		var labelValues []string
		if strings.Contains(spec.Name, "_vec") {
			labelValues = []string{"a", "b", "c"}
		}

		var methods []string
		switch spec.Type {
		default:
			t.Fatalf("Invalid metric handler type: %+v", spec)
		case "counter", "counter_vec":
			methods = []string{"inc", "add"}
		case "gauge", "gauge_vec":
			methods = []string{"set", "inc", "dec", "add", "sub", "set_to_current_time"}
		case "histogram", "histogram_vec", "summary", "summary_vec":
			methods = []string{"observe"}
		}

		if len(methods) == 0 {
			t.Fatalf("No methods found for spec: %+v", spec)
		}

		for _, method := range methods {
			m := Metric{
				Name:        spec.Name,
				Method:      method,
				Value:       1.0,
				LabelValues: labelValues,
			}
			if err := registry.Handle(&m); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestMetrics2Fail(t *testing.T) {
	SetTestLogger()
	specs := getTestSpecs(t, 2)

	if len(specs) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(specs))
	}

	registry := NewRegistry()
	for _, spec := range specs {
		if err := registry.Register(spec); err != nil {
			t.Fatal(err)
		}
	}

	for _, spec := range specs {
		var labelValues []string
		if strings.Contains(spec.Name, "_vec") {
			labelValues = []string{"a", "b", "c", "d"}
		}

		var methods []string
		switch spec.Type {
		default:
			t.Fatalf("Invalid metric handler type: %+v", spec)
		case "counter", "counter_vec":
			methods = []string{"inc", "add"}
		case "gauge", "gauge_vec":
			methods = []string{"set", "inc", "dec", "add", "sub", "set_to_current_time"}
		case "histogram", "histogram_vec", "summary", "summary_vec":
			methods = []string{"observe"}
		}

		if len(methods) == 0 {
			t.Fatalf("No methods found for spec: %+v", spec)
		}

		for _, method := range methods {
			m := Metric{
				Name:        spec.Name,
				Method:      method,
				Value:       1.0,
				LabelValues: labelValues,
			}
			err := registry.Handle(&m)
			if strings.Contains(spec.Name, "_vec") {
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

func TestMetrics3Rereg(t *testing.T) {
	SetTestLogger()
	specs := getTestSpecs(t, 3)
	specsUpdate := getTestSpecs(t, 4)

	if len(specs) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(specs))
	}

	if len(specsUpdate) != 10 {
		t.Errorf("Expected 10 metric specs, but got %d", len(specsUpdate))
	}

	registry := NewRegistry()

	// register all of specs
	for _, spec := range specs {
		if err := registry.Register(spec); err != nil {
			t.Fatal(err)
		}
	}

	names := registry.Names()

	// save specs[3] for later testing
	mySpec := specs[3]

	// emulate a USR1 with some of specsUpdate
	specs[3] = specsUpdate[3]
	specs[7] = specsUpdate[7]

	// register all of specsUpdate
	newNames := []string{}
	for idx, spec := range specs {
		err := registry.Register(spec)
		if err != nil {
			newNames = append(newNames, spec.Name)
		}
		if idx != 3 && idx != 7 {
			if err == nil {
				t.Fatalf("Expected spec %d to throw error, but did not", idx)
			}
		} else {
			if err != nil {
				t.Fatalf("Did not expect spec %d to throw error, but it did", idx)
			}
		}
	}

	unreg := sliceSubStr(names, newNames)

	if len(unreg) != 2 {
		t.Fatalf("Expected 2 specs to be unregistered, but got %d", len(unreg))
	}

	if !sliceContainsStr(unreg, mySpec.Name) {
		t.Fatal("%s should be getting unregistered", mySpec.Name)
	}

	for _, name := range unreg {
		if err := registry.Unregister(name); err != nil {
			t.Fatal(err)
		}
	}

	// modify a spec
	mySpec.Labels = []string{"Now", "for", "something", "completely", "different"}
	if err := registry.Register(mySpec); err == nil {
		t.Fatal("Expected re-reg of spec to throw error, but did not.")
	}
}
