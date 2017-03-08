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
)

var (
	logCloser io.WriteCloser
	logger    *log.Logger
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
			return fmt.Errorf("Error opening log file (%s): %s", file, err)
		}
		logger = log.New(logCloser, "", log.LstdFlags)
	}
	return nil
}

func LoadSpecs(file string) ([]*MetricSpec, error) {
	var (
		specs []*MetricSpec
		err   error
	)

	specsFile, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		return specs, err
	}
	defer specsFile.Close()

	return ReadSpecs(specsFile)
}

func ReadSpecs(r io.Reader) ([]*MetricSpec, error) {
	var result []*MetricSpec

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

func DataReader(ln net.Listener, workers int, metricCh chan<- *Metric) {
	dataCh := make(chan []byte)
	for i := 0; i < workers; i++ {
		go DataParser(dataCh, metricCh)
	}

	logger.Println("Starting listening on socket")
	for {
		// accept a connection
		c, err := ln.Accept()
		if err != nil {
			logger.Println(err)
			continue
		}

		var buf bytes.Buffer
		io.Copy(&buf, c)
		dataCh <- buf.Bytes()
		c.Close()
	}
	logger.Println("Ending listening on socket")
}

func DataParser(dataCh <-chan []byte, metricCh chan<- *Metric) {
	for {
		var metrics []Metric
		data := <-dataCh
		err := json.Unmarshal(data, &metrics)
		if err != nil {
			logger.Println(err)
			continue
		}
		for _, metric := range metrics {
			metricCh <- &metric
		}
	}
}

func DataProcessor(registry Registry, metricCh <-chan *Metric, doneCh <-chan bool) {
	logger.Println("Starting processing data")
	for {
		select {
		case metric := <-metricCh:
			if err := registry.Handle(metric); err != nil {
				logger.Println(err)
				continue
			}
		case <-doneCh:
			logger.Println("Ending processing data")
			return
		}
	}
}
