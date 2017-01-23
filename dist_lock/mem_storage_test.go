package dlock

import (
	"testing"
	"time"
)

func TestOpenClose(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	if ms.closed {
		t.Fatal("Expecting *ms be opened")
	}

	ms.Close()
	if !ms.closed {
		t.Fatal("*ms must be closed now")
	}
}

func TestCreateGet(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	r0 := &Record{"K", "val", 123, 0}
	ms.Create(r0)
	if len(ms.data) != 1 {
		t.Fatal("*ms must contain one element")
	}

	_, err := ms.Create(r0)
	if !CheckError(err, DLErrAlreadyExists) {
		t.Fatal("K must be already exist")
	}

	r, _ := ms.Get("K")
	if r == nil || r.Key != "K" || r.Ttl != 0 || r.Value != "val" || r.Version != 1 {
		t.Fatalf("Wrong record r0=%v, and found r=%v", r0, r)
	}

	_, err = ms.Get("k")
	if !CheckError(err, DLErrNotFound) {
		t.Fatal("Should not found ")
	}

	ms.Close()
	if len(ms.data) != 0 {
		t.Fatal("*ms must be clear")
	}

	_, err = ms.Get("k")
	if !CheckError(err, DLErrClosed) {
		t.Fatal("Should report already closed error")
	}
}

func TestImmutableGet(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	rVal := &Record{"rval", "val", 0, 0}
	r0 := &Record{"K", rVal, 123, 0}
	r, _ := ms.Create(r0)
	if len(ms.data) != 1 {
		t.Fatal("*ms must contain one element")
	}

	rVal1 := r.Value.(*Record)
	if rVal1.Key != "rval" {
		t.Fatal("We read something wrong ", rVal1)
	}
	rVal1.Key = "rval2"

	r, _ = ms.Get("K")
	if r.Value.(*Record).Key != "rval" {
		t.Fatal("We read something wrong val=", r.Value.(*Record))
	}

	ms.Close()
}

func TestTtl(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	r0 := &Record{"K", "val", 123, 50 * time.Millisecond}
	ms.Create(r0)
	<-time.After(time.Millisecond)

	r, _ := ms.Get("K")
	if r.Ttl > 50*time.Millisecond || r.Ttl < 1 {
		t.Fatalf("Wrong ttl for r=%v", r)
	}

	<-time.After(r.Ttl + 10*time.Millisecond)
	r, err := ms.Get("K")
	if !CheckError(err, DLErrNotFound) {
		t.Fatalf("Should Disappear r=%v, ms=%v", r, ms)
	}

	ms.Close()
}

func _TestSweep(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)

	if ms.sweep() < cMaxTime.Sub(time.Now()) {
		t.Fatal("for 0 sized array sleep shoul return maxTime")
	}

	r0 := &Record{"K", "val", 123, 50 * time.Millisecond}
	ms.Create(r0)

	if ms.sweep() > 50*time.Millisecond {
		t.Fatal("Expecting something aroung 50ms")
	}

	ms.Close()
	if ms.sweep() != 0 {
		t.Fatal("After close sweep should be 0")
	}
}

func TestCasByVersion(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	r0 := &Record{"K", "val", 123, 0}
	r, err := ms.Create(r0)
	if err != nil {
		t.Fatal("K must be created, but err=", err)
	}
	if r.Version != 1 {
		t.Fatalf("K version must be 1, but it is %v", r)
	}

	r.Ttl = time.Minute
	r, err = ms.CasByVersion(r)
	if r.Version != 2 {
		t.Fatalf("K version must be 2, but it is %v", r)
	}
	if err != nil {
		t.Fatal("CAS must work well err=", err)
	}
	if ms.sweepTime.After(time.Now().Add(time.Minute)) {
		t.Fatal("Sleep time is not adjusted")
	}
	if r.Ttl > time.Minute {
		t.Fatal("Ttl is not properly adjusted. It should be less than a minute")
	}

	r2, err := ms.CasByVersion(r0)
	if !CheckError(err, DLErrWrongVersion) {
		t.Fatal("Should report about wrong version err=", err)
	}
	if r2.Version != r.Version {
		t.Fatalf("K version (%v) must be same as in %v", r2, r)
	}

	ms.Close()
	_, err = ms.CasByVersion(r0)
	if !CheckError(err, DLErrClosed) {
		t.Fatal("Should report that already closed.")
	}
}

func TestWaitVersionChange(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	r0 := &Record{"K", "val", 123, 0}
	r, _ := ms.Create(r0)

	start := time.Now()
	ms.WaitVersionChange("K", r.Version, time.Millisecond*10)
	if time.Now().Sub(start) < time.Millisecond*10 {
		t.Fatal("Expecting to have timeout 10ms")
	}

	start = time.Now()

	go func() {
		// be sure we reached WaitVersion
		msr := ms.data["K"]
		for len(msr.waitChs) == 0 {
			time.After(time.Millisecond)
		}

		r.Value = "val2"
		ms.CasByVersion(r)
	}()

	r, _ = ms.WaitVersionChange("K", r.Version, time.Second)
	if time.Now().Sub(start) >= time.Second {
		t.Fatal("Expecting not to have 1 second timeout")
	}
	if r.Value != "val2" || r.Version != 2 {
		t.Fatal("Expecting updated version and value")
	}

	msr := ms.data["K"]
	if len(msr.waitChs) != 0 {
		t.Fatal("Expecting no opened channels in the record")
	}

	start = time.Now()

	go func() {
		// be sure we reached WaitVersion
		msr := ms.data["K"]
		for len(msr.waitChs) == 0 {
			time.After(time.Millisecond)
		}
		ms.Delete(r)
	}()

	r, err := ms.WaitVersionChange("K", r.Version, time.Second)
	if time.Now().Sub(start) >= time.Second {
		t.Fatal("Expecting not to have 1 second timeout")
	}
	if r != nil || !CheckError(err, DLErrNotFound) {
		t.Fatal("The value is deleted, why not nil?")
	}

	ms.Close()

}
