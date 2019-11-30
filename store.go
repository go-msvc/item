package item

import (
	"fmt"
	"sync"
)

//New creates an item store
func New(tmpl interface{}) IStore {
	return nil
}

//IStore stores items
type IStore interface {
}

//IStoreConfig stores items
type IStoreConfig interface {
	New() IStore
}

//RegisterStore is called for each implementation
func RegisterStore(name string, config IStoreConfig) {
	if len(name) == 0 {
		panic("RegisterStore(name=\"\")")
	}

	storeMutex.Lock()
	defer storeMutex.Unlock()
	if _, ok := storeConfigByName[name]; ok {
		panic(fmt.Sprintf("Duplicate name RegisterStore(name=\"%s\")", name))
	}

	storeConfigByName[name] = config
} //RegisterStore()

var (
	storeMutex        sync.Mutex
	storeConfigByName map[string]IStoreConfig
)
