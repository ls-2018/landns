package landns

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/landns/lib-landns/logger"
	"github.com/miekg/dns"
	"go.etcd.io/etcd/clientv3"
)

func init() {
	clientv3.SetLogger(logger.GRPCLogger{
		Fields: logger.Fields{"zone": "dynamic", "resolver": "EtcdResolver"},
	})
}

// EtcdResolver is one implements of DynamicResolver using etcd.
type EtcdResolver struct {
	client *clientv3.Client

	Timeout time.Duration
	Prefix  string
}

// NewEtcdResolver is constructor of EtcdResolver.
func NewEtcdResolver(endpoints []string, prefix string, timeout time.Duration, metrics *Metrics) (*EtcdResolver, error) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
	})
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to connect etcd"}
	}

	return &EtcdResolver{
		client:  c,
		Timeout: timeout,
		Prefix:  prefix,
	}, nil
}

// String is description string getter.
func (er *EtcdResolver) String() string {
	return fmt.Sprintf("EtcdResolver%s", er.client.Endpoints())
}

func (er *EtcdResolver) makeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), er.Timeout)
}

func (er *EtcdResolver) getKey(r DynamicRecord) string {
	if r.ID == nil {
		return fmt.Sprintf("%s/records%s", er.Prefix, r.Record.GetName().ToPath())
	}

	return fmt.Sprintf("%s/records%s/%d", er.Prefix, r.Record.GetName().ToPath(), *r.ID)
}

func (er *EtcdResolver) makeKey(ctx context.Context, r DynamicRecord) (DynamicRecord, string, error) {
	resp, err := er.client.Get(ctx, er.Prefix+"/lastID")
	if err != nil {
		return r, "", Error{TypeExternalError, err, "failed to get last ID"}
	}

	id := 0
	if len(resp.Kvs) > 0 {
		id, err = strconv.Atoi(string(resp.Kvs[0].Value))
		if err != nil {
			return r, "", Error{TypeInternalError, err, "failed to parse last ID"}
		}
	}
	id++
	r.ID = &id

	if _, err := er.client.Put(ctx, er.Prefix+"/lastID", strconv.Itoa(*r.ID)); err != nil {
		return r, "", Error{TypeExternalError, err, "failed to put last ID"}
	}

	return r, er.getKey(r), nil
}

func (er *EtcdResolver) findKey(ctx context.Context, r DynamicRecord, withTTL bool) (DynamicRecord, string, error) {
	resp, err := er.client.Get(ctx, er.getKey(r), clientv3.WithPrefix())
	if err != nil {
		return DynamicRecord{}, "", Error{TypeExternalError, err, "failed to get records"}
	}
	var r2 VolatileRecord
	for _, x := range resp.Kvs {
		if err := r2.UnmarshalText(x.Value); err != nil {
			return DynamicRecord{}, "", Error{TypeInternalError, err, "failed to parse record"}
		}

		rec, err := r2.Record()
		if err != nil {
			return DynamicRecord{}, "", Error{TypeInternalError, err, "failed to parse record"}
		}

		if withTTL && r.Record.String() != rec.String() {
			continue
		}
		if !withTTL && r.Record.WithoutTTL() != rec.WithoutTTL() {
			continue
		}

		id, err := er.getIDbyKey(x.Key)
		if err != nil {
			return DynamicRecord{}, "", err
		}
		r.ID = &id

		return r, string(x.Key), nil
	}

	r.ID = nil
	return r, "", nil
}

func (er *EtcdResolver) getIDbyKey(key []byte) (int, error) {
	ks := bytes.Split(key, []byte{'/'})

	i, err := strconv.Atoi(string(ks[len(ks)-1]))
	return i, wrapError(err, TypeInternalError, "failed to parse record ID")
}

func (er *EtcdResolver) readResponses(resp *clientv3.GetResponse) (DynamicRecordSet, error) {
	rs := make(DynamicRecordSet, 0, len(resp.Kvs))

	for _, r := range resp.Kvs {
		var vr VolatileRecord
		err := vr.UnmarshalText(r.Value)
		if err != nil {
			if e, ok := err.(Error); ok && e.Type == TypeExpirationError {
				continue
			} else {
				return nil, err
			}
		}

		var dr DynamicRecord
		dr.Record, err = vr.Record()
		if err != nil {
			return nil, err
		}

		dr.Volatile = vr.Expire.Unix() > 0

		id, err := er.getIDbyKey(r.Key)
		if err != nil {
			return nil, err
		}
		dr.ID = &id

		rs = append(rs, dr)
	}

	return rs, nil
}

func (er *EtcdResolver) dropRecord(ctx context.Context, r DynamicRecord) error {
	_, key, err := er.findKey(ctx, r, true)
	if err != nil {
		return err
	}

	if key != "" {
		if _, err = er.client.Delete(ctx, key); err != nil {
			return Error{TypeExternalError, err, "failed to delete record"}
		}
	}

	if r.Record.GetQtype() != dns.TypeA && r.Record.GetQtype() != dns.TypeAAAA {
		return nil
	}

	reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
	if err != nil {
		return Error{TypeExternalError, err, "failed to make reverse address"}
	}
	return er.dropRecord(ctx, DynamicRecord{
		Record: PtrRecord{
			Name:   Domain(reverse),
			TTL:    r.Record.GetTTL(),
			Domain: r.Record.GetName(),
		},
		Volatile: r.Volatile,
	})
}

func (er *EtcdResolver) insertSingleRecord(ctx context.Context, r DynamicRecord) error {
	r, key, err := er.findKey(ctx, r, false)
	if err != nil {
		return err
	}

	if key == "" {
		r, key, err = er.makeKey(ctx, r)
		if err != nil {
			return err
		}
	}

	options := []clientv3.OpOption{}
	if r.Volatile {
		resp, err := er.client.Grant(ctx, int64(r.Record.GetTTL()))
		if err != nil {
			return Error{TypeExternalError, err, "failed to grant TTL"}
		}
		options = append(options, clientv3.WithLease(resp.ID))
	}

	vr, err := r.VolatileRecord()
	if err != nil {
		return err
	}
	value, err := vr.MarshalText()
	if err != nil {
		return err
	}

	if _, err := er.client.Put(ctx, key, string(value), options...); err != nil {
		return Error{TypeExternalError, err, "failed to put record"}
	}

	return nil
}

func (er *EtcdResolver) insertRecord(ctx context.Context, r DynamicRecord) error {
	if err := er.insertSingleRecord(ctx, r); err != nil {
		return err
	}

	if r.Record.GetQtype() == dns.TypeA || r.Record.GetQtype() == dns.TypeAAAA {
		reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
		if err != nil {
			return Error{TypeExternalError, err, "failed to make reverse address"}
		}

		return er.insertSingleRecord(ctx, DynamicRecord{
			Record: PtrRecord{
				Name:   Domain(reverse),
				TTL:    r.Record.GetTTL(),
				Domain: r.Record.GetName(),
			},
		})
	}

	return nil
}

// SetRecords is DynamicRecord setter.
func (er *EtcdResolver) SetRecords(rs DynamicRecordSet) error {
	ctx, cancel := er.makeContext()
	defer cancel()

	for _, r := range rs {
		var err error

		if r.Disabled {
			err = er.dropRecord(ctx, r)
		} else {
			err = er.insertRecord(ctx, r)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// Records is DynamicRecord getter.
func (er *EtcdResolver) Records() (DynamicRecordSet, error) {
	ctx, cancel := er.makeContext()
	defer cancel()

	resp, err := er.client.Get(ctx, er.Prefix+"/records/", clientv3.WithPrefix())
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to get records"}
	}

	return er.readResponses(resp)
}

// SearchRecords is search records by domain prefix.
func (er *EtcdResolver) SearchRecords(d Domain) (DynamicRecordSet, error) {
	ctx, cancel := er.makeContext()
	defer cancel()

	resp, err := er.client.Get(ctx, er.Prefix+"/records"+d.ToPath(), clientv3.WithPrefix())
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to get records"}
	}

	return er.readResponses(resp)
}

func compileGlob(glob string) (func(string) bool, error) {
	for _, x := range []struct {
		From string
		To   string
	}{
		{`\`, `\\`},
		{`.`, `\.`},
		{`+`, `\+`},
		{`[`, `\[`},
		{`]`, `\]`},
		{`(`, `\(`},
		{`)`, `\)`},
		{`^`, `\^`},
		{`$`, `\$`},
		{`*`, `.*`},
	} {
		glob = strings.ReplaceAll(glob, x.From, x.To)
	}

	re, err := regexp.Compile("^" + glob + "$")
	if err != nil {
		return nil, Error{TypeInternalError, err, "failed to parse glob"}
	}

	return re.MatchString, nil
}

// GlobRecords is search records by glob string.
func (er *EtcdResolver) GlobRecords(glob string) (DynamicRecordSet, error) {
	check, err := compileGlob(glob)
	if err != nil {
		return nil, err
	}

	rs, err := er.Records()
	if err != nil {
		return nil, err
	}

	result := make(DynamicRecordSet, 0, len(rs))
	for _, r := range rs {
		if check(r.Record.GetName().String()) {
			result = append(result, r)
		}
	}
	return result, nil
}

// GetRecord is get record by id.
func (er *EtcdResolver) GetRecord(id int) (DynamicRecordSet, error) {
	rs, err := er.Records()
	if err != nil {
		return nil, err
	}

	for _, r := range rs {
		if *r.ID == id {
			return DynamicRecordSet{r}, nil
		}
	}
	return DynamicRecordSet{}, nil
}

// RemoveRecord is remove record by id.
func (er *EtcdResolver) RemoveRecord(id int) error {
	ctx, cancel := er.makeContext()
	defer cancel()

	rs, err := er.Records()
	if err != nil {
		return err
	}

	for _, r := range rs {
		if *r.ID == id {
			_, err = er.client.Delete(ctx, er.getKey(r))
			return wrapError(err, TypeExternalError, "failed to delete record")
		}
	}
	return ErrNoSuchRecord
}

// RecursionAvailable is always returns `false`.
func (er *EtcdResolver) RecursionAvailable() bool {
	return false
}

// Close is disconnector from etcd server.
func (er *EtcdResolver) Close() error {
	return wrapError(er.client.Close(), TypeExternalError, "failed to close etcd connection")
}

// Resolve is resolver using etcd.
func (er *EtcdResolver) Resolve(w ResponseWriter, r Request) error {
	name := Domain(r.Name)

	rs, err := er.SearchRecords(name)
	if err != nil {
		return err
	}

	for _, rec := range rs {
		if rec.Record.GetName() == name && rec.Record.GetQtype() == r.Qtype {
			if err := w.Add(rec.Record); err != nil {
				return err
			}
		}
	}

	return nil
}
