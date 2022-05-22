package client_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/macrat/landns/client/go-client"
	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
)

func GetAPIAddress() (*url.URL, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	tb := new(testutil.DummyTB)

	client, _ := testutil.StartServer(ctx, tb, false)

	return client.Endpoint, cancel
}

func Example() {
	apiAddress, closer := GetAPIAddress() // Start Landns server for test. (this is debug function)
	defer closer()

	c := client.New(apiAddress)

	rs, err := landns.NewDynamicRecordSet(`
	example.com. 100 IN A 127.0.0.1
	example.com. 200 IN A 127.0.0.2
`)
	if err != nil {
		panic(err.Error())
	}

	err = c.Set(rs) // Register records.
	if err != nil {
		panic(err.Error())
	}

	rs, err = c.Get() // Get all records.
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("all:")
	fmt.Print(rs)
	fmt.Println()

	rs, err = c.Glob("*.com") // Get records that ends with ".com".
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("glob:")
	fmt.Print(rs)
	fmt.Println()

	// Output:
	// all:
	// example.com. 100 IN A 127.0.0.1 ; ID:1
	// 1.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:2
	// example.com. 200 IN A 127.0.0.2 ; ID:3
	// 2.0.0.127.in-addr.arpa. 200 IN PTR example.com. ; ID:4
	//
	// glob:
	// example.com. 100 IN A 127.0.0.1 ; ID:1
	// example.com. 200 IN A 127.0.0.2 ; ID:3
}

func TestAPIClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, _ := testutil.StartServer(ctx, t, false)

	rs, err := landns.NewDynamicRecordSet(`a.example.com. 42 IN A 127.0.0.1 ; ID:1
1.0.0.127.in-addr.arpa. 42 IN PTR a.example.com. ; ID:2
b.example.com. 100 IN A 127.1.2.3 ; ID:3
3.2.1.127.in-addr.arpa. 100 IN PTR b.example.com. ; ID:4`)
	if err != nil {
		t.Fatalf("failed to parse records: %s", err)
	}

	if err := client.Set(rs); err != nil {
		t.Fatalf("failed to set records: %s", err)
	}

	if resp, err := client.Get(); err != nil {
		t.Fatalf("failed to get records: %s", err)
	} else if resp.String() != rs.String() {
		t.Fatalf("unexpected get response:\nexpect:\n%s\nbut got:\n%s", rs, resp)
	}

	expect := "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 100 IN A 127.1.2.3 ; ID:3\n"
	if resp, err := client.Glob("*.example.com"); err != nil {
		t.Fatalf("failed to glob records: %s", err)
	} else if resp.String() != expect {
		t.Fatalf("unexpected glob response:\nexpect:\n%s\nbut got:\n%s", expect, resp)
	}

	if err := client.Remove(2); err != nil {
		t.Fatalf("failed to remove records: %s", err)
	}

	expect = "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 100 IN A 127.1.2.3 ; ID:3\n3.2.1.127.in-addr.arpa. 100 IN PTR b.example.com. ; ID:4\n"
	if resp, err := client.Get(); err != nil {
		t.Fatalf("failed to glob records: %s", err)
	} else if resp.String() != expect {
		t.Fatalf("unexpected glob response:\nexpect:\n%s\nbut got:\n%s", expect, resp)
	}
}
