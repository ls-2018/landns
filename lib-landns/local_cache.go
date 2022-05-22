package landns

import (
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// LocalCache is in-memory cache manager for Resolver.
type LocalCache struct {
	mutex    sync.Mutex
	entries  map[uint16]map[Domain][]VolatileRecord
	invoke   chan struct{}
	closer   chan struct{}
	upstream Resolver
	metrics  *Metrics
}

// NewLocalCache is constructor of LocalCache.
//
// LocalCache will start background goroutine. So you have to ensure to call LocalCache.Close.
func NewLocalCache(upstream Resolver, metrics *Metrics) *LocalCache {
	lc := &LocalCache{
		entries:  make(map[uint16]map[Domain][]VolatileRecord),
		invoke:   make(chan struct{}, 100),
		closer:   make(chan struct{}),
		upstream: upstream,
		metrics:  metrics,
	}

	for _, t := range []uint16{dns.TypeA, dns.TypeNS, dns.TypeCNAME, dns.TypePTR, dns.TypeMX, dns.TypeTXT, dns.TypeAAAA, dns.TypeSRV} {
		lc.entries[t] = make(map[Domain][]VolatileRecord)
	}

	go lc.manage()

	return lc
}

// String is getter to description string.
func (lc *LocalCache) String() string {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	domains := make(map[Domain]struct{})
	records := 0
	for _, xs := range lc.entries {
		for name, x := range xs {
			domains[name] = struct{}{}
			records += len(x)
		}
	}

	return fmt.Sprintf("LocalCache[%d domains %d records]", len(domains), records)
}

// Close is closer to LocalCache.
func (lc *LocalCache) Close() error {
	close(lc.closer)
	close(lc.invoke)
	return nil
}

func (lc *LocalCache) manageTask() (next time.Duration) {
	next = 10 * time.Second

	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	for _, domains := range lc.entries {
		for name, entries := range domains {
			sweep := false

			for _, entry := range entries {
				delta := time.Until(entry.Expire)
				if delta < 1 {
					sweep = true
					break
				} else if next > delta {
					next = delta
				}
			}

			if sweep {
				delete(domains, name)
			}
		}
	}

	return next
}

func (lc *LocalCache) manage() {
	for {
		lc.manageTask()

		select {
		case <-time.After(lc.manageTask()):
		case <-lc.invoke:
		case <-lc.closer:
			return
		}
	}
}

func (lc *LocalCache) add(r Record) error {
	if r.GetTTL() == 0 {
		return nil
	}

	rr, err := r.ToRR()
	if err != nil {
		return err
	}

	if _, ok := lc.entries[r.GetQtype()][r.GetName()]; !ok {
		lc.entries[r.GetQtype()][r.GetName()] = []VolatileRecord{
			{rr, time.Now().Add(time.Duration(r.GetTTL()) * time.Second)},
		}
	} else {
		lc.entries[r.GetQtype()][r.GetName()] = append(
			lc.entries[r.GetQtype()][r.GetName()],
			VolatileRecord{rr, time.Now().Add(time.Duration(r.GetTTL()) * time.Second)},
		)
	}

	lc.invoke <- struct{}{}

	return nil
}

func (lc *LocalCache) resolveFromUpstream(w ResponseWriter, r Request) error {
	lc.metrics.CacheMiss(r)

	wh := ResponseWriterHook{
		Writer: w,
		OnAdd:  lc.add,
	}

	return lc.upstream.Resolve(wh, r)
}

func (lc *LocalCache) resolveFromCache(w ResponseWriter, r Request, records []VolatileRecord) error {
	lc.metrics.CacheHit(r)

	w.SetNoAuthoritative()

	for _, cache := range records {
		record, err := cache.Record()
		if err != nil {
			return err
		}

		if err := w.Add(record); err != nil {
			return err
		}
	}

	return nil
}

// Resolve is resolver using cache or the upstream resolver.
func (lc *LocalCache) Resolve(w ResponseWriter, r Request) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	records, ok := lc.entries[r.Qtype][Domain(r.Name)]
	if !ok {
		return lc.resolveFromUpstream(w, r)
	}

	for _, cache := range records {
		if time.Until(cache.Expire) < 1 {
			delete(lc.entries[r.Qtype], Domain(r.Name))
			return lc.resolveFromUpstream(w, r)
		}
	}

	return lc.resolveFromCache(w, r, records)
}

// RecursionAvailable is returns same as upstream.
func (lc *LocalCache) RecursionAvailable() bool {
	return lc.upstream.RecursionAvailable()
}
