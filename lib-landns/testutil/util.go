package testutil

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

const (
	// PortMin is minimum port number for FindEmptyPort.
	PortMin = 49152

	// PortMax is maximum port number for FindEmptyPort.
	PortMax = 65535
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// FindEmptyPort is find unused TCP port.
func FindEmptyPort() int {
	for {
		port := rand.Intn(PortMax-PortMin+1) + PortMin
		l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			l.Close()
			return port
		}
	}
}
