package gorivets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// TODO: describe behaviour
var gmap struct {
	gm   map[string]interface{}
	file *os.File
	lock sync.Mutex
}

func GMapGet(key string) (interface{}, bool) {
	gmap.lock.Lock()
	defer gmap.lock.Unlock()

	checkGMap()
	v, ok := gmap.gm[key]
	return v, ok
}

func GMapPut(key string, value interface{}) (interface{}, bool) {
	gmap.lock.Lock()
	defer gmap.lock.Unlock()

	checkGMap()
	v, ok := gmap.gm[key]
	gmap.gm[key] = value
	return v, ok
}

func GMapDelete(key string) interface{} {
	gmap.lock.Lock()
	defer gmap.lock.Unlock()

	checkGMap()
	v := gmap.gm[key]
	delete(gmap.gm, key)

	if len(gmap.gm) == 0 {
		deleteGMap()
	}

	return v
}

func GMapShutdown() {
	gmap.lock.Lock()
	defer gmap.lock.Unlock()

	deleteGMap()
}

func deleteGMap() {
	if gmap.file != nil {
		fn := gmap.file.Name()
		gmap.file.Close()
		os.Remove(fn)
		gmap.file = nil
		gmap.gm = nil
	}
}

func checkGMap() {
	if gmap.gm != nil {
		return
	}

	createGMapFile()
	gmap.gm = make(map[string]interface{})
}

func createGMapFile() {
	dir := os.TempDir()
	fileName := fmt.Sprintf("gorivets.globalMap.%d", os.Getpid())
	name := filepath.Join(dir, fileName)
	var err error
	gmap.file, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		panic(errors.New("\nCould not create global map: the file \"" +
			name + "\" already exists. It could happen by one of the following reasons:\n" +
			"\t1) different versions of gorivets are linked into the application - check vendors folders of the application dependencies\n" +
			"\t2) the file left from a previously run process, which has been crashed - delete it and re-run the application.\n"))
	}
}
