package dlock

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"
)

type ms_record struct {
	key        string
	value      string
	version    int
	elemType   reflect.Type
	changeTime time.Time
	ttl        time.Duration
	waitChs    []chan bool
}

type mem_storage struct {
	data      map[string]*ms_record
	lock      sync.Mutex
	closed    bool
	sweepTime time.Time
	sweepCh   chan bool
}

func NewMemStorage() Storage {
	ms := &mem_storage{sweepCh: make(chan bool, 1)}

	go func() {
		for !ms.closed {
			sleepTimeout := time.Minute
			select {
			case _, ok := <-ms.sweepCh:
				if !ok {
					return
				}
			case <-time.After(sleepTimeout):
			}
			ms.sweep()
		}
	}()

	return ms
}

func (ms *mem_storage) sweep() {
	maxTime := time.Unix(1<<63-1, 0)
	now := time.Now()

	// if ms.sweepTime == maxTime OR it is before now, we will walk through
	// the list, otherwise go out of here...
	if maxTime.After(ms.sweepTime) && now.Before(ms.sweepTime) {
		return
	}

	ms.lock.Lock()
	defer ms.lock.Unlock()

	minTime := maxTime
	for k, msr := range ms.data {
		if msr.ttl == 0 {
			continue
		}
		expTime := msr.changeTime.Add(msr.ttl)

		if expTime.Before(now) {
			delete(ms.data, k)
			continue
		}

		if expTime.Before(minTime) {
			minTime = expTime
		}
	}

	ms.sweepTime = minTime
}

func (ms *mem_storage) Create(record *Record) (*Record, error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	r, ok := ms.data[record.Key]
	if ok {
		return toRecord(r), Error(DLErrAlreadyExists)
	}

	r = to_ms_record(record)
	r.version = 1
	ms.data[record.Key] = r

	return toRecord(r), nil
}

func (ms *mem_storage) Get(key string) (*Record, error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	r, ok := ms.data[key]
	if !ok {
		return nil, Error(DLErrNotFound)
	}

	return toRecord(r), nil
}

func (ms *mem_storage) CasByVersion(record *Record) (*Record, error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	r, ok := ms.data[record.Key]
	if !ok {
		return nil, Error(DLErrNotFound)
	}

	if r.version != record.Version {
		return toRecord(r), Error(DLErrWrongVersion)
	}

	r1 := to_ms_record(record)
	r1.version = r.version + 1
	ms.data[record.Key] = r1
	r.notifyChans()
	return toRecord(r1), nil
}

func (ms *mem_storage) Delete(record *Record) (*Record, error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	r, ok := ms.data[record.Key]
	if !ok {
		return nil, Error(DLErrNotFound)
	}

	if r.version != record.Version {
		return toRecord(r), Error(DLErrWrongVersion)
	}

	delete(ms.data, record.Key)
	r.notifyChans()
	return nil, nil
}

func (ms *mem_storage) WaitVersionChange(key string, version int, timeout time.Duration) (*Record, error) {
	ch, err := ms.newChan(key, version)
	if err != nil {
		r, _ := ms.Get(key)
		return r, err
	}

	select {
	case <-ch:
	case <-time.After(timeout):
		ms.dropChan(key, ch)
	}
	return ms.Get(key)
}

func (ms *mem_storage) Close() {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	for _, msr := range ms.data {
		msr.notifyChans()
	}

	ms.closed = true
}

func (ms *mem_storage) newChan(key string, version int) (chan bool, error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if ms.closed {
		return nil, Error(DLErrClosed)
	}

	r, ok := ms.data[key]
	if !ok {
		return nil, Error(DLErrNotFound)
	}

	if r.version != version {
		return nil, Error(DLErrWrongVersion)
	}

	ch := make(chan bool)
	r.waitChs = append(r.waitChs, ch)
	return ch, nil
}

func (ms *mem_storage) dropChan(key string, ch chan bool) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	msr, _ := ms.data[key]
	if msr == nil {
		return
	}

	for i, c := range msr.waitChs {
		if c == ch {
			l := len(msr.waitChs)
			msr.waitChs[l-1], msr.waitChs[i] = msr.waitChs[i], msr.waitChs[l-1]
			msr.waitChs = msr.waitChs[:l-1]
			return
		}
	}
}

func (msr *ms_record) notifyChans() {
	for _, ch := range msr.waitChs {
		ch <- true
		close(ch)
	}
	msr.waitChs = make([]chan bool, 0)
}

func toRecord(r *ms_record) *Record {
	val := reflect.New(r.elemType)
	json.NewDecoder(strings.NewReader(r.value)).Decode(&val)
	ttl := r.ttl - time.Now().Sub(r.changeTime)
	if r.ttl == 0 {
		ttl = 0
	} else if ttl <= 0 {
		// We will not provide 0 or a value <0 cause 0 means infinite, but
		// r.ttl > 0 means that we have a timeout...
		ttl = 1
	}
	return &Record{Key: r.key, Value: val, Version: r.version, Ttl: ttl}
}

func to_ms_record(r *Record) *ms_record {
	et := reflect.ValueOf(r.Value).Elem().Type()
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(r.Value)
	var ttl time.Duration = 0
	if r.Ttl > 0 {
		ttl = r.Ttl
	}
	return &ms_record{r.Key, buf.String(), r.Version, et, time.Now(),
		ttl, make([]chan bool, 0)}
}
