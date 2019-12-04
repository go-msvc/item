package store

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-msvc/errors"
)

//Register is called for each implementation
func Register(name string, config IStoreConfig) {
	if len(name) == 0 {
		panic("RegisterStore(name=\"\")")
	}

	storeMutex.Lock()
	defer storeMutex.Unlock()
	if _, ok := storeConfigByName[name]; ok {
		panic(fmt.Sprintf("Duplicate name RegisterStore(name=\"%s\")", name))
	}

	storeConfigByName[name] = config
} //Register()

//IStoreConfig stores items
type IStoreConfig interface {
	New(itemName string, itemType reflect.Type) (IStore, error)
}

var (
	storeMutex        sync.Mutex
	storeConfigByName = make(map[string]IStoreConfig)
)

//New creates an item store
func New(tmpl interface{}) (IStore, error) {
	return nil, fmt.Errorf("NYI")
}

//MustNew ...
func MustNew(tmpl interface{}) IStore {
	s, err := New(tmpl)
	if err != nil {
		panic(err)
	}
	return s
}

//IStore stores items
type IStore interface {
	//singulat name of an item in this store (e.g. "user", "subscription", etc...)
	Name() string
	Type() reflect.Type

	Add(v interface{}) (info ItemInfo, err error) //creates rev=1

	//Get the latest revision
	Get(id ID) (v interface{}, info ItemInfo, err error)

	//Get info of the latest revision
	GetInfo(id ID) (info ItemInfo, err error) //faster than Get(), only return header

	//GetBy arbitrary key fields
	GetBy(max int, key map[string]interface{}) (items []interface{}, info []ItemInfo, err error)

	//update to create a new revision (id will not change)
	Upd(id ID, v interface{}) (info ItemInfo, err error)

	//Get a specific revision
	//GetRev(id ID, rev int) (v interface{}, info ItemInfo, err error)

	Del(id ID) error
}

//ValidateUserType ...
func ValidateUserType(t reflect.Type) error {
	if t.Kind() != reflect.Struct {
		return errors.Errorf("%v is %v but should be %v", t, t.Kind(), reflect.Struct)
	}
	if t.NumField() < 1 {
		return errors.Errorf("%v is struct without fields", t)
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			return errors.Errorf("%v has anonymous field", t)
		}
		if len(f.PkgPath) > 0 {
			return errors.Errorf("%v.%s is unexported field", t, f.Name)
		}
	}
	return nil
} //ValidateUserType()

//ItemInfo ...
type ItemInfo struct {
	ID        ID
	Rev       int
	Timestamp time.Time
	UserID    ID
}
