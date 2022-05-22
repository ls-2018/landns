package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/alecthomas/kingpin"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/logger"
)

func loadStatisResolvers(files []string) (resolver landns.ResolverSet, err error) {
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer (*file).Close()

		config, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		r, err := landns.NewSimpleResolverFromConfig(config)
		if err != nil {
			return nil, err
		}

		resolver = append(resolver, r)
	}

	return resolver, nil
}

type service struct {
	App       *kingpin.Application
	Start     func(context.Context) error
	Stop      func() error
	DNSListen *net.TCPAddr
	APIListen *net.TCPAddr
}

func makeServer(args []string) (*service, error) {
	app := kingpin.New("landns", "A DNS server for developers for home use.")
	configFiles := app.Flag("config", "Path to static-zone configuration file.").Short('c').PlaceHolder("PATH").ExistingFiles()
	sqlitePath := app.Flag("sqlite", "Path to dynamic-zone sqlite3 database path. In default, dynamic-zone will not save to disk.").Short('s').PlaceHolder("PATH").String()
	etcdAddrs := app.Flag("etcd", "Address to dynamic-zone etcd database server. (e.g. localhost:2379)").PlaceHolder("ADDRESS").Strings()
	etcdPrefix := app.Flag("etcd-prefix", "Prefix of etcd records.").Default("/landns").String()
	etcdTimeout := app.Flag("etcd-timeout", "Timeout for etcd connection.").Default("100ms").Duration()
	apiListen := app.Flag("api-listen", "Address for API and metrics.").Short('l').Default(":9353").TCP()
	dnsListen := app.Flag("dns-listen", "Address for listen.").Short('L').Default(":53").TCP()
	dnsProtocol := app.Flag("dns-protocol", "Protocol for listen.").Default("udp").Enum("udp", "tcp")
	upstreams := app.Flag("upstream", "Upstream DNS server for recursive resolve. (e.g. 8.8.8.8:53)").Short('u').PlaceHolder("ADDRESS").TCPList()
	upstreamTimeout := app.Flag("upstream-timeout", "Timeout for recursive resolve.").Default("100ms").Duration()
	cacheDisabled := app.Flag("disable-cache", "Disable cache for recursive resolve.").Bool()
	redisAddr := app.Flag("redis", "Address of Redis server for sharing recursive resolver's cache. (e.g. 127.0.0.1:6379)").PlaceHolder("ADDRESS").TCP()
	redisPassword := app.Flag("redis-password", "Password of Redis server.").PlaceHolder("PASSWORD").String()
	redisDatabase := app.Flag("redis-database", "Database ID of Redis server.").PlaceHolder("ID").Int()
	metricsNamespace := app.Flag("metrics-namespace", "Namespace of prometheus metrics.").Default("landns").String()
	verbose := app.Flag("verbose", "Show verbose logs.").Short('v').Bool()
	pprof := app.Flag("enable-pprof", "Enable pprof API.").Bool()

	_, err := app.Parse(args)
	if err != nil {
		return nil, err
	}

	level := logger.WarnLevel
	if *verbose {
		level = logger.InfoLevel
	}
	logger.SetLogger(logger.New(os.Stdout, level))

	metrics := landns.NewMetrics(*metricsNamespace)

	resolvers, err := loadStatisResolvers(*configFiles)
	if err != nil {
		return nil, fmt.Errorf("static-zone: %s", err)
	}

	var dynamicResolver landns.DynamicResolver
	if *sqlitePath != "" && len(*etcdAddrs) != 0 {
		return nil, fmt.Errorf("dynamic-zone: can't use both of sqlite and etcd")
	} else if len(*etcdAddrs) > 0 {
		dynamicResolver, err = landns.NewEtcdResolver(*etcdAddrs, *etcdPrefix, *etcdTimeout, metrics)
	} else {
		if *sqlitePath == "" {
			*sqlitePath = ":memory:"
		}
		dynamicResolver, err = landns.NewSqliteResolver(*sqlitePath, metrics)
	}
	if err != nil {
		return nil, fmt.Errorf("dynamic-zone: %s", err)
	}
	resolvers = append(resolvers, dynamicResolver)

	var resolver landns.Resolver = resolvers
	if len(*upstreams) > 0 {
		us := make([]*net.UDPAddr, len(*upstreams))
		for i, u := range *upstreams {
			us[i] = &net.UDPAddr{
				IP:   u.IP,
				Port: u.Port,
				Zone: u.Zone,
			}
		}
		var forwardResolver landns.Resolver = landns.NewForwardResolver(us, *upstreamTimeout, metrics)
		if !*cacheDisabled {
			if *redisAddr != nil {
				forwardResolver, err = landns.NewRedisCache(*redisAddr, *redisDatabase, *redisPassword, forwardResolver, metrics)
				if err != nil {
					return nil, fmt.Errorf("recursive: Redis cache: %s", err)
				}
			} else {
				forwardResolver = landns.NewLocalCache(forwardResolver, metrics)
			}
		}
		resolver = landns.AlternateResolver{resolver, forwardResolver}
	}

	server := landns.Server{
		Metrics:         metrics,
		DynamicResolver: dynamicResolver,
		Resolvers:       resolver,
		DebugMode:       *pprof,
	}
	return &service{
		App: app,
		Start: func(ctx context.Context) error {
			return server.ListenAndServe(
				ctx,
				*apiListen,
				&net.UDPAddr{IP: (*dnsListen).IP, Port: (*dnsListen).Port},
				*dnsProtocol,
			)
		},
		Stop:      resolver.Close,
		DNSListen: *dnsListen,
		APIListen: *apiListen,
	}, nil
}

func main() {
	service, err := makeServer(os.Args[1:])
	if err != nil {
		logger.Fatal("failed to start server", logger.Fields{"reason": err})
	}
	defer func() {
		if err := service.Stop(); err != nil {
			logger.Fatal("failed to stop server", logger.Fields{"reason": err})
		}
	}()

	logger.Info("starting API server", logger.Fields{"address": service.APIListen})
	logger.Info("starting DNS server", logger.Fields{"address": service.DNSListen})
	if err := service.Start(context.Background()); err != nil {
		logger.Fatal("failed to running server", logger.Fields{"reason": err})
	}
}
