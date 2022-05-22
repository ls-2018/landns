package landns

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Domain is the type for domain.
type Domain string

// String is converter to Domain to string.
//
// String will convert to FQDN.
func (d Domain) String() string {
	return dns.Fqdn(string(d))
}

// Normalized is normalizer for make FQDN.
func (d Domain) Normalized() Domain {
	return Domain(d.String())
}

// Validate is validator to domain.
func (d Domain) Validate() error {
	if len(string(d)) == 0 {
		return newError(TypeArgumentError, nil, "invalid domain: %#v", string(d))
	}
	if _, ok := dns.IsDomainName(string(d)); !ok {
		return newError(TypeArgumentError, nil, "invalid domain: %#v", string(d))
	}
	return nil
}

// UnmarshalText is parse text from bytes.
func (d *Domain) UnmarshalText(text []byte) error {
	*d = Domain(dns.Fqdn(string(text)))

	return d.Validate()
}

// MarshalText is make bytes text.
func (d Domain) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// ToPath is make reversed path string like /com/example.
func (d Domain) ToPath() string {
	labels := dns.SplitDomainName(d.String())
	rev := make([]string, len(labels))
	for i, x := range labels {
		rev[len(labels)-i-1] = x
	}
	return "/" + strings.Join(rev, "/")
}

// Record is the record entry of DNS.
type Record interface {
	fmt.Stringer

	GetQtype() uint16      // Get query type like dns.TypeA of package github.com/miekg/dns.
	GetName() Domain       // Get domain name of the Record.
	GetTTL() uint32        // Get TTL for the Record.
	ToRR() (dns.RR, error) // Get response record for dns.Msg of package github.com/miekg/dns.
	Validate() error       // Validation Record and returns error if invalid.
	WithoutTTL() string    // Get record string that replaced TTL number with "0".
}

// NewRecord is make new Record from query string.
func NewRecord(str string) (Record, error) {
	rr, err := dns.NewRR(str)
	if err != nil {
		return nil, Error{TypeArgumentError, err, "failed to parse record"}
	}

	return NewRecordFromRR(rr)
}

// NewRecordWithTTL is make new Record with TTL.
func NewRecordWithTTL(str string, ttl uint32) (Record, error) {
	rr, err := dns.NewRR(str)
	if err != nil {
		return nil, Error{TypeArgumentError, err, "failed to parse record"}
	}

	rr.Header().Ttl = ttl

	return NewRecordFromRR(rr)
}

// NewRecordWithExpire is make new Record from query string with expire time.
func NewRecordWithExpire(str string, expire time.Time) (Record, error) {
	if expire.Before(time.Now()) {
		return nil, newError(TypeExpirationError, nil, "expire can't be past time: %s", expire)
	}

	return NewRecordWithTTL(str, uint32(math.Round(time.Until(expire).Seconds())))
}

// NewRecordFromRR is make new Record from dns.RR of package github.com/miekg/dns.
func NewRecordFromRR(rr dns.RR) (Record, error) {
	switch x := rr.(type) {
	case *dns.A:
		return AddressRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Address: x.A}, nil
	case *dns.AAAA:
		return AddressRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Address: x.AAAA}, nil
	case *dns.NS:
		return NsRecord{Name: Domain(x.Hdr.Name), Target: Domain(x.Ns)}, nil
	case *dns.CNAME:
		return CnameRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Target: Domain(x.Target)}, nil
	case *dns.PTR:
		return PtrRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Domain: Domain(x.Ptr)}, nil
	case *dns.MX:
		return MxRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Preference: x.Preference, Target: Domain(x.Mx)}, nil
	case *dns.TXT:
		return TxtRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Text: x.Txt[0]}, nil
	case *dns.SRV:
		return SrvRecord{
			Name:     Domain(x.Hdr.Name),
			TTL:      x.Hdr.Ttl,
			Priority: x.Priority,
			Weight:   x.Weight,
			Port:     x.Port,
			Target:   Domain(x.Target),
		}, nil
	default:
		return nil, newError(TypeArgumentError, nil, "unsupported record type: %d", rr.Header().Rrtype)
	}
}

// AddressRecord is the Record of A or AAAA.
type AddressRecord struct {
	Name    Domain
	TTL     uint32
	Address net.IP
}

// IsV4 is checker for guess that which of IPv4 (A record) or IPv6 (AAAA record).
func (r AddressRecord) IsV4() bool {
	return r.Address.To4() != nil
}

// String is make record string.
func (r AddressRecord) String() string {
	qtype := "A"
	if !r.IsV4() {
		qtype = "AAAA"
	}
	return fmt.Sprintf("%s %d IN %s %s", r.Name, r.TTL, qtype, r.Address)
}

// WithoutTTL is make record string but mask TTL number.
func (r AddressRecord) WithoutTTL() string {
	qtype := "A"
	if !r.IsV4() {
		qtype = "AAAA"
	}

	return fmt.Sprintf("%s 0 IN %s %s", r.Name, qtype, r.Address)
}

// GetName is getter to name of record.
func (r AddressRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r AddressRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r AddressRecord) GetQtype() uint16 {
	if r.IsV4() {
		return dns.TypeA
	}

	return dns.TypeAAAA
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r AddressRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r AddressRecord) Validate() error {
	return r.Name.Validate()
}

// NsRecord is the Record of NS.
type NsRecord struct {
	Name   Domain
	Target Domain
}

// Strings is make record string.
func (r NsRecord) String() string {
	return fmt.Sprintf("%s IN NS %s", r.Name, r.Target)
}

// WithoutTTL is make record string but mask TTL number.
func (r NsRecord) WithoutTTL() string {
	return r.String()
}

// GetName is getter to name of record.
func (r NsRecord) GetName() Domain {
	return r.Name
}

// GetTTL is always returns 0 because NS record has no TTL.
func (r NsRecord) GetTTL() uint32 {
	return 0
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r NsRecord) GetQtype() uint16 {
	return dns.TypeNS
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r NsRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r NsRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Target.Validate()
}

// CnameRecord is the Record of CNAME.
type CnameRecord struct {
	Name   Domain
	TTL    uint32
	Target Domain
}

// String is make record string.
func (r CnameRecord) String() string {
	return fmt.Sprintf("%s %d IN CNAME %s", r.Name, r.TTL, r.Target)
}

// WithoutTTL is make record string but mask TTL number.
func (r CnameRecord) WithoutTTL() string {
	return fmt.Sprintf("%s 0 IN CNAME %s", r.Name, r.Target)
}

// GetName is getter to name of record.
func (r CnameRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r CnameRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r CnameRecord) GetQtype() uint16 {
	return dns.TypeCNAME
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r CnameRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r CnameRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Target.Validate()
}

// PtrRecord is the Record of PTR.
type PtrRecord struct {
	Name   Domain
	TTL    uint32
	Domain Domain
}

// String is make record string.
func (r PtrRecord) String() string {
	return fmt.Sprintf("%s %d IN PTR %s", r.Name, r.TTL, r.Domain)
}

// WithoutTTL is make record string but mask TTL number.
func (r PtrRecord) WithoutTTL() string {
	return fmt.Sprintf("%s 0 IN PTR %s", r.Name, r.Domain)
}

// GetName is getter to name of record.
func (r PtrRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r PtrRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r PtrRecord) GetQtype() uint16 {
	return dns.TypePTR
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r PtrRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r PtrRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Domain.Validate()
}

// MxRecord is the Record of CNAME.
type MxRecord struct {
	Name       Domain
	TTL        uint32
	Preference uint16
	Target     Domain
}

// String is make record string.
func (r MxRecord) String() string {
	return fmt.Sprintf("%s %d IN MX %d %s", r.Name, r.TTL, r.Preference, r.Target)
}

// WithoutTTL is make record string but mask TTL number.
func (r MxRecord) WithoutTTL() string {
	return fmt.Sprintf("%s 0 IN MX %d %s", r.Name, r.Preference, r.Target)
}

// GetName is getter to name of record.
func (r MxRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r MxRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r MxRecord) GetQtype() uint16 {
	return dns.TypeMX
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r MxRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r MxRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Target.Validate()
}

// TxtRecord is the Record of TXT.
type TxtRecord struct {
	Name Domain
	TTL  uint32
	Text string
}

// String is make record string.
func (r TxtRecord) String() string {
	return fmt.Sprintf("%s %d IN TXT \"%s\"", r.Name, r.TTL, r.Text)
}

// WithoutTTL is make record string but mask TTL number.
func (r TxtRecord) WithoutTTL() string {
	return fmt.Sprintf("%s 0 IN TXT \"%s\"", r.Name, r.Text)
}

// GetName is getter to name of record.
func (r TxtRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r TxtRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r TxtRecord) GetQtype() uint16 {
	return dns.TypeTXT
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r TxtRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r TxtRecord) Validate() error {
	return r.Name.Validate()
}

// SrvRecord is the Record of SRV.
type SrvRecord struct {
	Name     Domain
	TTL      uint32
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   Domain
}

// String is make record string.
func (r SrvRecord) String() string {
	return fmt.Sprintf(
		"%s %d IN SRV %d %d %d %s",
		r.Name,
		r.TTL,
		r.Priority,
		r.Weight,
		r.Port,
		r.Target,
	)
}

// WithoutTTL is make record string but mask TTL number.
func (r SrvRecord) WithoutTTL() string {
	return fmt.Sprintf(
		"%s 0 IN SRV %d %d %d %s",
		r.Name,
		r.Priority,
		r.Weight,
		r.Port,
		r.Target,
	)
}

// GetName is getter to name of record.
func (r SrvRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r SrvRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r SrvRecord) GetQtype() uint16 {
	return dns.TypeSRV
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r SrvRecord) ToRR() (dns.RR, error) {
	rr, err := dns.NewRR(r.String())
	return rr, wrapError(err, TypeInternalError, "failed to convert to RR")
}

// Validate is validator of record.
func (r SrvRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	if r.Port == 0 {
		return newError(TypeArgumentError, nil, "invalid port: %d", uint16(r.Port))
	}
	return r.Target.Validate()
}

// VolatileRecord is record value that has expire datetime.
type VolatileRecord struct {
	RR     dns.RR
	Expire time.Time
}

// NewVolatileRecord will parse record text and make new VolatileRecord.
func NewVolatileRecord(record string) (VolatileRecord, error) {
	var r VolatileRecord
	return r, r.UnmarshalText([]byte(record))
}

// Record is Record getter.
func (r VolatileRecord) Record() (Record, error) {
	if r.Expire.Unix() > 0 {
		ttl := math.Round(time.Until(r.Expire).Seconds())
		if ttl < 0 {
			return nil, newError(TypeExpirationError, nil, "this record is already expired: %s", r.Expire)
		}

		r.RR.Header().Ttl = uint32(ttl)
	}

	return NewRecordFromRR(r.RR)
}

// String is get printable string.
func (r VolatileRecord) String() string {
	text, _ := r.MarshalText()
	return string(text)
}

// UnmarshalText is unmarshal VolatileRecord from text.
func (r *VolatileRecord) UnmarshalText(text []byte) error {
	if bytes.Contains(text, []byte("\n")) {
		return ErrMultiLineDynamicRecord
	}
	text = bytes.TrimSpace(text)

	xs := bytes.SplitN(text, []byte(";"), 2)

	var err error
	r.RR, err = dns.NewRR(string(xs[0]))
	if err != nil {
		return Error{TypeInternalError, err, "failed to parse record"}
	}

	r.Expire = time.Unix(0, 0)

	if len(xs) == 2 {
		i, err := strconv.ParseInt(string(bytes.TrimSpace(xs[1])), 10, 64)
		if err != nil {
			return Error{TypeInternalError, err, "failed to parse record"}
		}
		r.Expire = time.Unix(i, 0)

		if r.Expire.Before(time.Now()) {
			return newError(TypeExpirationError, nil, "failed to parse record: expire can't be past time: %s", r.Expire)
		}
	}

	return nil
}

// MarshalText is marshal VolatileRecord to text.
func (r VolatileRecord) MarshalText() ([]byte, error) {
	rec, err := r.Record()
	if err != nil {
		return nil, err
	}

	if r.Expire.Unix() > 0 {
		return []byte(fmt.Sprintf("%s ; %d", rec, r.Expire.Unix())), nil
	}

	return []byte(rec.String()), nil
}
