package landns_test

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func AssertResolve(t testing.TB, resolver landns.Resolver, request landns.Request, authoritative bool, responses ...string) {
	t.Helper()

	resp := testutil.NewDummyResponseWriter()
	if err := resolver.Resolve(resp, request); err != nil {
		t.Errorf("%s <- %s: failed to resolve: %v", resolver, request, err.Error())
		return
	}

	if resp.Authoritative != authoritative {
		t.Errorf("%s <- %s: unexpected authoritive of response: expected %v but got %v", resolver, request, authoritative, resp.Authoritative)
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return strings.Compare(resp.Records[i].String(), resp.Records[j].String()) == 1
	})
	sort.Slice(responses, func(i, j int) bool {
		return strings.Compare(responses[i], responses[j]) == 1
	})

	ok := len(resp.Records) == len(responses)

	if ok {
		for i := range responses {
			if resp.Records[i].String() != responses[i] {
				ok = false
				break
			}
		}
	}

	if !ok {
		msg := fmt.Sprintf("%s <- %s: unexpected resolve response:\nexpected:\n", resolver, request)
		for _, x := range responses {
			msg += "\t" + x + "\n"
		}
		msg += "but got:\n"
		for _, x := range resp.Records {
			msg += "\t" + x.String() + "\n"
		}
		t.Errorf("%s", msg)
	}
}

func AssertExchange(t *testing.T, addr *net.UDPAddr, question []dns.Question, expect ...string) {
	t.Helper()

	msg := &dns.Msg{
		MsgHdr:   dns.MsgHdr{Id: dns.Id()},
		Question: question,
	}

	in, err := dns.Exchange(msg, addr.String())
	if err != nil {
		t.Errorf("%s: failed to resolve: %s", addr, err)
		return
	}

	ok := len(in.Answer) == len(expect)

	if ok {
		for i := range expect {
			if in.Answer[i].String() != expect[i] {
				ok = false
				break
			}
		}
	}

	if !ok {
		msg := "%s: unexpected answer:\nexpected:\n"
		for _, x := range expect {
			msg += x + "\n"
		}
		msg += "\nbut got:\n"
		for _, x := range in.Answer {
			msg += x.String() + "\n"
		}
		t.Errorf(msg, addr)
	}
}

func AssertDynamicRecordSet(t testing.TB, expect []string, got landns.DynamicRecordSet) {
	t.Helper()

	sort.Slice(expect, func(i, j int) bool {
		return strings.Compare(expect[i], expect[j]) == 1
	})
	sort.Slice(got, func(i, j int) bool {
		return strings.Compare(got[i].String(), got[j].String()) == 1
	})

	ok := len(expect) == len(got)
	if ok {
		for i := range got {
			if got[i].String() != expect[i] {
				ok = false
			}
		}
	}
	if !ok {
		txt := "unexpected entries:\nexpected:\n"
		for _, t := range expect {
			txt += "\t" + t + "\n"
		}
		txt += "\nbut got:\n"

		for _, r := range got {
			txt += "\t" + r.String() + "\n"
		}
		t.Errorf(txt)
	}
}

func CheckRecursionAvailable(t testing.TB, makeResolver func([]landns.Resolver) landns.Resolver) {
	t.Helper()

	recursionResolver := testutil.DummyResolver{Error: false, Recursion: true}
	nonRecursionResolver := testutil.DummyResolver{Error: false, Recursion: false}

	resolver := makeResolver([]landns.Resolver{nonRecursionResolver, recursionResolver, nonRecursionResolver})
	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}

	resolver = makeResolver([]landns.Resolver{nonRecursionResolver, nonRecursionResolver})
	if resolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}
}

func ParallelResolveTest(t testing.TB, resolver landns.Resolver) {
	errors := make([]chan string, 5)
	loop := 100

	for i := range errors {
		errors[i] = make(chan string)
		go func(ch chan string) {
			defer close(ch)
			for i := 0; i < loop; i++ {
				err := resolver.Resolve(testutil.EmptyResponseWriter{}, landns.NewRequest("example.com.", dns.TypeA, false))
				if err != nil {
					ch <- err.Error()
				}
			}
		}(errors[i])
	}

	errorSet := make(map[string]struct{})
	errorCount := 0
	for _, ch := range errors {
		for err := range ch {
			errorSet[err] = struct{}{}
			errorCount++
		}
	}
	errorList := []string{}
	for err := range errorSet {
		errorList = append(errorList, err)
	}
	if len(errorList) != 0 {
		t.Errorf("pararell resolve errors: rate: %.2f%%\n%s", float64(errorCount)*100/float64(loop*len(errors)), strings.Join(errorList, "\n"))
	}
}
