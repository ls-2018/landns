package testutil_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns/testutil"
)

func TestStartServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, addr := testutil.StartServer(ctx, t, false)

	if addr == nil {
		t.Errorf("invalid address: %v", addr)
	}

	_, err := client.Get()
	if err != nil {
		t.Errorf("failed to connect server: %s", err)
	}

	cancel()
	time.Sleep(10 * time.Millisecond)

	_, err = client.Get()
	if ok, e := regexp.MatchString(`Get \"http://[^: ]+:[0-9]+/api/v1\": dial tcp [^: ]+:[0-9]+: connect: connection refused`, err.Error()); e != nil || !ok {
		t.Errorf("failed to connect server: %s", err)
	}
}
