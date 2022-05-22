package landns_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/logger"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/integration"
)

func init() {
	clientv3.SetLogger(logger.GRPCLogger{
		Logger: logger.New(os.Stderr, logger.ErrorLevel),
		Fields: logger.Fields{"zone": "dynamic", "resolver": "EtcdResolver"},
	})
}

func CreateEtcdResolver(t testing.TB) (*landns.EtcdResolver, []string, func()) {
	clus := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 1, SkipCreatingClient: true})

	addrs := make([]string, len(clus.Members))
	for i, m := range clus.Members {
		addrs[i] = m.GRPCAddr()
	}

	resolver, err := landns.NewEtcdResolver(addrs, "/landns", time.Second, landns.NewMetrics("metrics"))
	if err != nil {
		t.Fatalf("failed to make etcd resolver: %s", err)
	}

	return resolver, addrs, func() {
		clus.Terminate(t)
		if err := resolver.Close(); err != nil {
			t.Errorf("failed to close: %s", err)
		}
	}
}

func TestEtcdResolver(t *testing.T) {
	t.Parallel()

	for _, tt := range DynamicResolverTests {
		tester := tt.Tester

		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			resolver, addrs, closer := CreateEtcdResolver(t)
			defer closer()

			name := fmt.Sprintf("EtcdResolver[%s]", addrs[0])
			if s := resolver.String(); s != name {
				t.Errorf(`unexpected string: expected %#v but got %#v`, name, s)
			}

			tester(t, resolver)
		})
	}

	t.Run("NewEtcdResolver_fail", func(t *testing.T) {
		t.Parallel()

		_, err := landns.NewEtcdResolver(nil, "/landns", time.Second, landns.NewMetrics("landns"))
		expect := "failed to connect etcd: etcdclient: no available endpoints"
		if err == nil {
			t.Errorf("expected error but got nil")
		} else if err.Error() != expect {
			t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", expect, err.Error())
		}
	})
}

func BenchmarkEtcdResolver(b *testing.B) {
	resolver, _, closer := CreateEtcdResolver(b)
	defer closer()

	DynamicResolverBenchmark(b, resolver)
}
