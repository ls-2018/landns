package testutil_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns/testutil"
)

func TestFindEmptyPort(t *testing.T) {
	t.Parallel()

	port := testutil.FindEmptyPort()
	if port == -1 {
		t.Errorf("failed to find empty port")
	} else if port < testutil.PortMin || testutil.PortMax < port {
		t.Errorf("unexpected range: %d", port)
	}
}
