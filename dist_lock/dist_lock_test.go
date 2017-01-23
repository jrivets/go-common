package dlock

import (
	"testing"
	"time"

	"../../gorivets"
)

func TestStartStop(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	dlm := new_dlock_manager(ms, time.Millisecond)
	dlm.start()

	<-time.After(time.Millisecond)
	id := cKeyLockerId + dlm.lockerId
	r, err := ms.Get(id)
	if err != nil || r == nil {
		t.Fatal("At this moment we have to have a record to keep the instance alive")
	}
	if r.Ttl > time.Millisecond || r.Ttl < 1 {
		t.Fatal("Wrong Ttl: ", r)
	}

	dlm.stop()
	if !ms.closed {
		t.Fatal("Close() on storage must be called")
	}
}

func TestSimpleLocking(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	SetStorage(ms)
	lock := NewLock("val1")
	lock2 := NewLock("val2")

	_, err := ms.Get("val1")
	if !CheckError(err, DLErrNotFound) {
		t.Fatal("Should be no val1 record")
	}

	lock.Lock()
	_, err = ms.Get("val1")
	if err != nil {
		t.Fatal("Should be val1 record")
	}
	_, err = ms.Get("val2")
	if !CheckError(err, DLErrNotFound) {
		t.Fatal("Should be no val2 record")
	}

	lock2.Lock()
	_, err = ms.Get("val2")
	if err != nil {
		t.Fatal("Should be val2 record")
	}
	lock.Unlock()
	lock2.Unlock()

	_, err = ms.Get("val1")
	if !CheckError(err, DLErrNotFound) {
		t.Fatal("Should be no val1 record")
	}
	_, err = ms.Get("val2")
	if !CheckError(err, DLErrNotFound) {
		t.Fatal("Should be no val2 record")
	}

	Shutdown()
}

func TestRelockPanic(t *testing.T) {
	ms := NewMemStorage().(*mem_storage)
	SetStorage(ms)
	lock := NewLock("val1")

	lock.Lock()
	if gorivets.CheckPanic(func() { lock.Lock() }) == nil {
		t.Fatal("Should panicing")
	}
	Shutdown()
}
