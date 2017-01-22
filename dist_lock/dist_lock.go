package dlock

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/satori/go.uuid"
)

type (
	Record struct {
		Key     string
		Value   interface{}
		Version int

		// Time To live in nanoseconds. This value indicates how long the record
		// is going to be kept in the storage. Value 0 shows that Ttl is not
		// limited
		Ttl time.Duration
	}

	Storage interface {
		// Creates a new one, or returns existing one with DLErrAlreadyExists
		// error
		Create(record *Record) (*Record, error)

		// Retrieves the record by its key. It will return nil and an error,
		// which will indicate the reason why the operation was not succesful.
		Get(key string) (*Record, error)

		// Compare and set the record value if the record stored version is
		// same as in the provided record. The record version will be updated
		// too.
		//
		// Returns updated stored record or stored record and error if the
		// operation was not successful
		CasByVersion(record *Record) (*Record, error)

		// Tries to delete the record. Operation can fail if stored record
		// version is different than existing one. This case the stored version
		// is returned as well as the appropriate error. If the record is deleted
		// both results are nil(!)
		Delete(record *Record) (*Record, error)

		// Waits for the record version change. The version param contans an
		// expected version.
		WaitVersionChange(key string, version int, timeout time.Duration) (*Record, error)

		// Indicates that the storage is going to be closed and all blocked
		// calls (like WaitVersionChange) can over immediately
		Close()
	}

	Error int
)

const (
	// Errors
	DLErrAlreadyExists Error = 1
	DLErrNotFound      Error = 2
	DLErrWrongVersion  Error = 3
	DLErrClosed        Error = 4

	// dlock_manager states
	ST_STARTING = 1
	ST_STARTED
	ST_STOPPED

	// Reserved keys
	cKeyLockerId = "__locker_id__"

	// Default config settings
	cKeepAliveSec = 30
)

func CheckError(e error, expErr Error) bool {
	if e == nil {
		return false
	}
	err, ok := e.(Error)
	if !ok {
		return false
	}
	return err == expErr
}

func (e Error) Error() string {
	switch e {
	case DLErrAlreadyExists:
		return "Record with the key already exists"
	case DLErrNotFound:
		return "Record with the key is not found"
	case DLErrWrongVersion:
		return "Unexpected record version"
	}
	return ""
}

func NewLock(name string) sync.Locker {
	if name == cKeyLockerId {
		panic("Wrong name=" + name + ". It is reserved")
	}
	return &dlock{name}
}

func SetStorage(storage Storage) {
	dlmLock.Lock()
	defer dlmLock.Unlock()

	if dlm != nil {
		panic(errors.New("Wrong initialization cycle: dlm already initialized"))
	}

	dlm = &dlock_manager{storage: storage, state: ST_STARTING}
	dlm.cfgKeepAliveSec = cKeepAliveSec
	dlm.lockerId = uuid.NewV4().String()
	dlm.llocks = make(map[string]*local_lock)
	dlm.start()
}

func Shutdown() {
	dlmLock.Lock()
	defer dlmLock.Unlock()

	dlm.stop()
}

// ================================ Details ===================================
type dlock struct {
	name string
}

func (dl *dlock) Lock() {
	dlm.lockGlobal(dl.name)
}

func (dl *dlock) Unlock() {
	dlm.unlockGlobal(dl.name)
}

type dlock_manager struct {
	storage  Storage
	stopChan chan bool
	lock     sync.Mutex
	state    int
	lockerId string
	llocks   map[string]*local_lock

	//Config settings

	// Sets the instance storage record keep-alive timeout
	cfgKeepAliveSec int
}

type local_lock struct {
	cond    *sync.Cond
	counter int
}

// Stored to storage
type lock_info struct {
	owner string
}

var dlm *dlock_manager
var dlmLock sync.Mutex

func (dlm *dlock_manager) start() {
	dlm.lock.Lock()
	defer dlm.lock.Unlock()

	if dlm.state != ST_STARTING {
		panic(errors.New("Wrong dlock_manager state=" + strconv.Itoa(dlm.state)))
	}

	dlm.stopChan = make(chan bool, 1)
	dlm.state = ST_STARTED
	go func() {
		for {
			select {
			case <-dlm.stopChan:
				return
			case <-time.After(time.Second * time.Duration(dlm.cfgKeepAliveSec/2)):
				dlm.keepAlive()
			}
		}
	}()
}

func (dlm *dlock_manager) stop() {
	dlm.lock.Lock()
	defer dlm.lock.Unlock()

	if dlm.state != ST_STARTED {
		panic(errors.New("Wrong dlock_manager state=" + strconv.Itoa(dlm.state)))
	}

	// Lets all waiters know about finishing the game
	for _, ll := range dlm.llocks {
		ll.cond.Broadcast()
	}

	dlm.state = ST_STARTED
	dlm.stopChan <- true
	close(dlm.stopChan)
	dlm.storage.Close()
}

func (dlm *dlock_manager) keepAlive() {
	for {
		r := &Record{cKeyLockerId, true, 0, time.Duration(dlm.cfgKeepAliveSec) * time.Second}
		r, err := dlm.storage.Create(r)
		if err == nil {
			return
		}

		if CheckError(err, DLErrAlreadyExists) {
			for {
				r.Ttl = time.Duration(dlm.cfgKeepAliveSec) * time.Second
				r, err = dlm.storage.CasByVersion(r)

				if err == nil {
					return
				}

				if CheckError(err, DLErrNotFound) {
					break
				}

				if !CheckError(err, DLErrWrongVersion) {
					panic(err)
				}
			}
		}
	}
}

func (dlm *dlock_manager) lockGlobal(name string) {
	if !dlm.lockLocal(name) {
		panic("Could not lock distributed lock \"" + name + "\"locally")
	}

	li := &lock_info{dlm.lockerId}
	r, err := dlm.storage.Create(&Record{name, li, 0, 0})

	if err == nil {
		return
	}

	if !CheckError(err, DLErrAlreadyExists) {
		dlm.unlockGlobal(name)
		panic(err)
	}

	for {
		li = r.Value.(*lock_info)
		if dlm.isLockerIdValid(li.owner) {
			r, err = dlm.storage.WaitVersionChange(name, r.Version, time.Duration(dlm.cfgKeepAliveSec)*time.Second)
			if CheckError(err, DLErrClosed) {
				dlm.unlockGlobal(name)
				panic(err)
			}

			li = r.Value.(*lock_info)
		} else {
			li.owner = ""
		}

		if li.owner == "" {
			li.owner = dlm.lockerId
			r, err = dlm.storage.CasByVersion(r)
			if err == nil {
				return
			}
		}
	}
}

func (dlm *dlock_manager) unlockGlobal(name string) {
	r, err := dlm.storage.Get(name)

	for r != nil && !CheckError(err, DLErrNotFound) {
		li := r.Value.(*lock_info)
		if li.owner != dlm.lockerId {
			dlm.unlockLocal(name)
			panic(errors.New("FATAL internal error: unlocking object which is locked by other locker. " +
				"expected lockerId=" + dlm.lockerId + ", but returned one is " + li.owner))
		}

		r, err = dlm.storage.Delete(r)
	}

	dlm.unlockLocal(name)
}

func (dlm *dlock_manager) lockLocal(name string) bool {
	dlm.lock.Lock()
	defer dlm.lock.Unlock()

	if dlm.state != ST_STARTED {
		return false
	}

	ll := dlm.llocks[name]
	if ll == nil {
		ll = &local_lock{cond: sync.NewCond(&dlm.lock), counter: 0}
		dlm.llocks[name] = ll
	}
	ll.counter++

	for ll.counter > 1 {
		ll.cond.Wait()

		if dlm.state != ST_STARTED {
			dlm.unlockLocalUnsafe(name, ll)
			return false
		}
	}

	return true
}

func (dlm *dlock_manager) unlockLocal(name string) {
	dlm.lock.Lock()
	defer dlm.lock.Unlock()

	ll := dlm.llocks[name]
	dlm.unlockLocalUnsafe(name, ll)
	ll.cond.Signal()
}

func (dlm *dlock_manager) unlockLocalUnsafe(name string, ll *local_lock) {
	ll.counter--
	if ll.counter == 0 {
		delete(dlm.llocks, name)
	}
}

func (dlm *dlock_manager) isLockerIdValid(lockerId string) bool {
	if dlm.lockerId == lockerId {
		return true
	}

	_, err := dlm.storage.Get(lockerId)
	if CheckError(err, DLErrNotFound) {
		return false
	}
	return true
}
