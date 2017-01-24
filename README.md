# prom_multi_proc

Prometheus metrics aggregation for forking servers.

Listens on a unix socket and collects prometheus metrics, then exposes metrics via http
for prometheus scrape.

Addresses issue where a master process forks several workers and each worker collects
his own metrics. When prometheus server scrapes the master, it will return the metrics
from whichever worker serviced the request. This app allows all workers to write
metrics to a single socket, where the metrics can be aggregated.

App servers where this is applicable are unicorn and puma (ruby).

See https://github.com/prometheus/client_ruby/issues/9
for background information.

## Install

Download the [latest release](https://github.com/DripEmail/prom_multi_proc/releases), extract it,
and put it somewhere on your PATH.

or

```sh
$ go get github.com/DripEmail/prom_multi_proc
```

or

```sh
$ mkdir -p $GOPATH/src/github.com/DripEmail
$ cd $GOPATH/src/github.com/DripEmail
$ git clone git@github.com:DripEmail/prom_multi_proc.git
$ cd prom_multi_proc
$ go install
$ rehash
```

## Testing

```sh
$ cd $GOPATH/src/github.com/DripEmail/prom_multi_proc
$ go test -cover
```

## Releases

```sh
$ mkdir -p $GOPATH/src/github.com/DripEmail
$ cd $GOPATH/src/github.com/DripEmail
$ git clone git@github.com:DripEmail/prom_multi_proc.git
$ cd prom_multi_proc
$ make release
```

## Command-Line Options

```
λ prom_multi_proc -h
Usage of prom_multi_proc:
  -addr string
        Address to listen on for exposing prometheus metrics (default "0.0.0.0:9299")
  -log string
        Path to log file, will write to STDOUT if empty
  -metrics string
        Path to json file which contains metric definitions
  -path string
        Path to use for exposing prometheus metrics (default "/metrics")
  -socket string
        Path to unix socket to listen on for incoming metrics (default "/tmp/prom_multi_proc.sock")
  -v    Print version information and exit
```
