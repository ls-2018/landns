Landns docker-compose sample
============================

The sample of [Landns](https://github.com/macrat/landns), [prometheus](https://prometheus.io), and [grafana](http://grafana.com).


## Usage

Clone this repository,

``` shell
$ git clone https://github.com/macrat/landns
$ cd landns/docker-compose
```

And start it.

``` shell
$ docker-compose up -d
```

Then, use DNS. And see Grafana dashboard on [localhost:3000](http://localhost:3000) (ID and password is `admin`) or Prometheus console on [localhost:9090](http://localhost:9090).
