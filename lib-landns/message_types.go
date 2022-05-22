package landns

import (
	"fmt"

	"github.com/miekg/dns"
)

// QtypeToString is helper for dns type number to human readable string like "A" or "TXT".
func QtypeToString(qtype uint16) string {
	switch qtype {
	case dns.TypeA:
		return "A"
	case dns.TypeNS:
		return "NS"
	case dns.TypeCNAME:
		return "CNAME"
	case dns.TypePTR:
		return "PTR"
	case dns.TypeMX:
		return "MX"
	case dns.TypeTXT:
		return "TXT"
	case dns.TypeAAAA:
		return "AAAA"
	case dns.TypeSRV:
		return "SRV"
	default:
		return "UNKNOWN"
	}
}

// Request is structure of DNS question.
type Request struct {
	dns.Question

	RecursionDesired bool
}

// NewRequest is constructor for Request.
func NewRequest(name string, qtype uint16, recursionDesired bool) Request {
	return Request{dns.Question{Name: name, Qtype: qtype, Qclass: dns.ClassINET}, recursionDesired}
}

// QtypeString is getter to human readable record type.
func (req Request) QtypeString() string {
	return QtypeToString(req.Qtype)
}

// String is converter to query string.
func (req Request) String() string {
	return fmt.Sprintf(";%s IN %s", req.Name, req.QtypeString())
}

// ResponseWriter is interface for Resolver.
type ResponseWriter interface {
	Add(Record) error      // Add new record into response.
	IsAuthoritative() bool // Check current response is authoritative or not.
	SetNoAuthoritative()   // Set no authoritative.
}

// ResponseCallback is one implements of ResponseWriter for callback function.
type ResponseCallback struct {
	Callback      func(Record) error
	Authoritative bool
}

func NewResponseCallback(callback func(Record) error) *ResponseCallback {
	return &ResponseCallback{Callback: callback, Authoritative: true}
}

func (rc *ResponseCallback) Add(r Record) error {
	return rc.Callback(r)
}

func (rc *ResponseCallback) IsAuthoritative() bool {
	return rc.Authoritative
}

func (rc *ResponseCallback) SetNoAuthoritative() {
	rc.Authoritative = false
}

// ResponseWriterHook is a wrapper of ResponseWriter for hook events.
type ResponseWriterHook struct {
	Writer ResponseWriter
	OnAdd  func(Record) error
}

func (rh ResponseWriterHook) Add(r Record) error {
	if rh.OnAdd != nil {
		if err := rh.OnAdd(r); err != nil {
			return err
		}
	}
	return rh.Writer.Add(r)
}

func (rh ResponseWriterHook) IsAuthoritative() bool {
	return rh.Writer.IsAuthoritative()
}

func (rh ResponseWriterHook) SetNoAuthoritative() {
	rh.Writer.SetNoAuthoritative()
}

// MessageBuilder is one implements of ResponseWriter for make dns.Msg of package github.com/miekg/dns.
type MessageBuilder struct {
	request            *dns.Msg
	records            []dns.RR
	authoritative      bool
	recursionAvailable bool
}

func NewMessageBuilder(request *dns.Msg, recursionAvailable bool) *MessageBuilder {
	return &MessageBuilder{
		request:            request,
		records:            make([]dns.RR, 0, 10),
		authoritative:      true,
		recursionAvailable: recursionAvailable,
	}
}

func (mb *MessageBuilder) Add(r Record) error {
	rr, err := r.ToRR()
	if err != nil {
		return err
	}

	mb.records = append(mb.records, rr)
	return nil
}

func (mb *MessageBuilder) IsAuthoritative() bool {
	return mb.authoritative
}

func (mb *MessageBuilder) SetNoAuthoritative() {
	mb.authoritative = false
}

// Build is builder of dns.Msg.
func (mb *MessageBuilder) Build() *dns.Msg {
	msg := new(dns.Msg)
	msg.SetReply(mb.request)

	msg.Answer = dns.Dedup(mb.records, nil)

	msg.Authoritative = mb.authoritative
	msg.RecursionAvailable = mb.recursionAvailable

	return msg
}
