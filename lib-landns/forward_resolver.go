package landns

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

// ForwardResolver is recursion resolver.
type ForwardResolver struct {
	client *dns.Client

	Upstreams []*net.UDPAddr
	Metrics   *Metrics
}

// NewForwardResolver is make new ForwardResolver.
func NewForwardResolver(upstreams []*net.UDPAddr, timeout time.Duration, metrics *Metrics) ForwardResolver {
	return ForwardResolver{
		client: &dns.Client{
			Dialer: &net.Dialer{
				Timeout: timeout,
			},
		},
		Upstreams: upstreams,
		Metrics:   metrics,
	}
}

// Resolve is resolver using upstream DNS servers.
func (fr ForwardResolver) Resolve(w ResponseWriter, r Request) error {
	if !r.RecursionDesired {
		return nil
	}

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{
			{Name: r.Name, Qtype: r.Qtype, Qclass: r.Qclass},
		},
	}

	for _, upstream := range fr.Upstreams {
		in, rtt, err := fr.client.Exchange(msg, upstream.String())
		if err != nil {
			continue
		}
		fr.Metrics.UpstreamTime(rtt)
		for _, answer := range in.Answer {
			record, err := NewRecordFromRR(answer)
			if err != nil {
				return err
			}
			w.SetNoAuthoritative()
			if err := w.Add(record); err != nil {
				return err
			}
		}
		break
	}

	return nil
}

// RecursionAvailable is always returns true.
func (fr ForwardResolver) RecursionAvailable() bool {
	return true
}

// Close is closer to ForwardResolver.
func (fr ForwardResolver) Close() error {
	return nil
}
