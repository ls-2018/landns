package landns

import (
	"testing"
	"time"
)

func TestSqliteResolver_manageExpire(t *testing.T) {
	t.Parallel()

	resolver, err := NewSqliteResolver(":memory:", NewMetrics("landns"))
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err.Error())
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	recordsNum := func() int {
		resolver.mutex.Lock()
		defer resolver.mutex.Unlock()

		var num int
		if err := resolver.db.QueryRow("SELECT COUNT(name) FROM records").Scan(&num); err != nil {
			t.Fatalf("faield to get number of records: %s", err)
		}
		return num
	}

	close(resolver.closer)
	resolver.closer = make(chan struct{})
	go resolver.manageExpire(1 * time.Second)

	records, err := NewDynamicRecordSet(`
		fixed.example.com. 100 IN TXT "fixed"
		long.example.com. 100 IN TXT "long" ; Volatile
		short.example.com. 1 IN TXT "short" ; Volatile
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Fatalf("failed to set records: %s", err)
	}

	if num := recordsNum(); num != 3 {
		t.Errorf("unexpected number of records: expected 3 but got %d", num)
	}

	time.Sleep(3 * time.Second)

	if num := recordsNum(); num != 2 {
		t.Errorf("unexpected number of records: expected 2 but got %d", num)
	}
}
