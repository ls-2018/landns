Landns
======

![GitHub Actions](https://github.com/macrat/landns/workflows/Test%20and%20Build/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/macrat/landns/branch/master/graph/badge.svg)](https://codecov.io/gh/macrat/landns)
[![Go Report Card](https://goreportcard.com/badge/github.com/macrat/landns)](https://goreportcard.com/report/github.com/macrat/landns)
[![Docker Hub](https://img.shields.io/badge/container-Docker%20Hub-blue.svg?logo=docker&logoColor=white)](https://hub.docker.com/r/macrat/landns)
[![GitHub Container Registry](https://img.shields.io/badge/container-ghcr.io-blue.svg?logo=docker&logoColor=white)](https://github.com/users/macrat/packages/container/package/landns)
[![License](https://img.shields.io/github/license/macrat/landns)](https://github.com/macrat/landns/blob/master/LICENSE)

A DNS server for developers for home use.


## Features

- Serve addresses from the YAML style configuration file.

- Serve addresses from a [SQlite](https://www.sqlite.org/) or [etcd](https://etcd.io) database that operatable with REST API.

- Recursion resolve and caching addresses to local memory or [Redis server](https://redis.io).

- Built-in metrics exporter for [Prometheus](https://prometheus.io).


## How to use

### Install server

Please use `go get`.

``` shell
$ go get github.com/macrat/landns
```

Or, you can use docker.

``` shell
$ docker run -p 9353:9353/tcp -p 53:53/udp macrat/landns:latest
```

### Use as static DNS server

Make setting file like this.

``` yaml
ttl: 600

address:
  router.local: [192.168.1.1]
  servers.example.com:
    - 192.168.1.10
    - 192.168.1.11
    - 192.168.1.12

cname:
  gateway.local: [router.local]

text:
  message.local:
    - hello
    - world

services:
  example.com:
    - service: http
      port: 80
      proto: tcp    # optional (default: tcp)
      priority: 10  # optional (default: 0)
      weight: 5     # optional (default: 0)
      target: servers.example.com
    - service: ftp
      port: 21
      target: servers.example.com
```

And then, execute server.

``` shell
$ sudo landns --config path/to/config.yml
```

### Use as dynamic DNS server

First, execute server.

``` shell
$ sudo landns --sqlite path/to/database.db  # use SQlite database.

$ sudo landns --etcd 127.0.0.1:2379  # use etcd database.
```

Dynamic settings that set by REST API will store to specified database if given `--sqlite` or `--etcd` option.
REST API will work if not gven it, but settings will lose when the server stopped.

Then, operate records with API.

``` shell
$ curl http://localhost:9353/api/v1 -d 'www.example.com 600 IN A 192.168.1.1'
; 200: add:1 delete:0

$ curl http://localhost:9353/api/v1 -d 'ftp.example.com 600 IN CNAME www.example.com'
; 200: add:1 delete:0

$ curl http://localhost:9353/api/v1
www.example.com 600 IN A 192.168.1.1 ; ID:1
1.1.168.192.in-addr.arpa. 600 IN PTR www.example.com ; ID:2
ftp.example.com 600 IN CNAME www.example.com ; ID:3

$ curl http://localhost:9353/api/v1/suffix/com/example
www.example.com 600 IN A 192.168.1.1 ; ID:1
ftp.example.com 600 IN CNAME www.example.com ; ID:3

$ curl http://localhost:9353/api/v1/suffix/example.com
www.example.com 600 IN A 192.168.1.1 ; ID:1
ftp.example.com 600 IN CNAME www.example.com ; ID:3

$ curl http://localhost:9353/api/v1/glob/w*ample.com
www.example.com 600 IN A 192.168.1.1 ; ID:1
```

``` shell
$ cat config.zone
router.service. 600 IN A 192.168.1.1
gateway.service. 600 IN CNAME router.local.
alice.pc.local. 600 IN A 192.168.1.10

$ curl http://localhost:9353/api/v1 --data-binary @config.zone
; 200: add:3 delete:0

$ curl http://localhost:9353/api/v1
router.service. 600 IN A 192.168.1.1 ; ID:1
1.1.168.192.in-addr.arpa. 600 IN PTR router.service. ; ID:2
gateway.service. 600 IN CNAME router.local. ; ID:3
alice.pc.local. 600 IN A 192.168.1.10 ; ID:4
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5
```

There are 3 ways to remove records.

``` shell
$ curl http://localhost:9353/api/v1 -X DELETE -d 'router.service. 600 IN A 192.168.1.1 ; ID:1'  # Use DELETE method
; 200: add:0 delete:1

$ curl http://localhost:9353/api/v1
gateway.service. 600 IN CNAME router.local. ; ID:3
alice.pc.local. 600 IN A 192.168.1.10 ; ID:4
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5

$ curl http://localhost:9353/api/v1 -X POST -d ';gateway.service. 600 IN CNAME router.local. ; ID:3'  # Use comment style
; 200: add:0 delete:1

$ curl http://localhost:9353/api/v1
alice.pc.local. 600 IN A 192.168.1.10 ; ID:4
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5

$ curl http://localhost:9353/api/v1/id/4 -X DELETE  # Use DELETE method with ID
; 200: ok

$ curl http://localhost:9353/api/v1
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5
```

You can use variable.
`$TTL` will replace to `3600`, and `$ADDR` will replace to IP address of client.

``` shell
$ curl http://localhost:9353/api/v1 -d 'example.com. $TTL IN A $ADDR'  # Use variable.
; 200: add:1 delete:0

$ curl http://localhost:9353/api/v1
example.com. 3600 IN A 127.0.0.1 ; ID:1
1.0.0.127.in-addr.arpa. 3600 IN PTR example.com. ; ID:2
```

### Get metrics (with prometheus)

Landns serve metrics for Prometheus by default in port 9353.


### Use as library

Landns can use as a library like below.

``` golang
package main

import (
	"context"
	"net"

	"github.com/macrat/landns/lib-landns"
)

type Resolver struct {
	metrics *landns.Metrics
}

func (rs Resolver) Resolve(w landns.ResponseWriter, r landns.Request) error {
	w.Add(landns.AddressRecord{
		Name:    "test.local",
		TTL:     100,
		Address: net.ParseIP("127.1.2.3"),
	})
	return nil
}

func (rs Resolver) RecursionAvailable() bool {
	return false
}

func (rs Resolver) Close() error {
	return nil
}

func main() {
	metrics := landns.NewMetrics("test_dns")
	resolver := Resolver{metrics}
	server := landns.Server{
		Name:      "Test DNS", // The name for page title of metrics server.
		Metrics:   metrics,
		Resolvers: resolver,
		DebugMode: true,
	}
	server.ListenAndServe(
		context.Background(),
		&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8053},
		&net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 1053},
		"udp",
	)
}
```

Above code will behave DNS server for `test.local`, and metrics server.
