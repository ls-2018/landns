package landns

import (
	"github.com/macrat/landns/lib-landns/logger"
	"github.com/miekg/dns"
)

// Handler is the implements of dns.Handler of package github.com/miekg/dns.
type Handler struct {
	Resolver           Resolver
	Metrics            *Metrics
	RecursionAvailable bool
}

// NewHandler is constructor of Handler.
func NewHandler(resolver Resolver, metrics *Metrics) Handler {
	return Handler{
		Resolver:           resolver,
		Metrics:            metrics,
		RecursionAvailable: resolver.RecursionAvailable(),
	}
}

// ServeDNS is the method for resolve record.
func (h Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	end := h.Metrics.Start(r)

	req := Request{RecursionDesired: r.RecursionDesired}
	resp := NewMessageBuilder(r, h.RecursionAvailable)

	errored := false

	if r.Opcode == dns.OpcodeQuery {
		for _, q := range r.Question {
			req.Question = q

			if err := h.Resolver.Resolve(resp, req); err != nil {
				logger.Warn("failed to resolve", logger.Fields{"proto": "dns", "name": q.Name, "type": QtypeToString(q.Qtype), "reason": err})
				h.Metrics.Error(req, err)
				errored = true
			}
		}
	}

	msg := resp.Build()
	if err := w.WriteMsg(msg); err != nil {
		logger.Error("failed to write msg", nil)
	}
	end(msg)

	if !errored && len(msg.Answer) == 0 && len(msg.Question) > 0 {
		q := msg.Question[0]
		logger.Info("not found", logger.Fields{"proto": "dns", "name": q.Name, "type": QtypeToString(q.Qtype)})
	}
}
