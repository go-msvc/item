package memory

import (
	"reflect"
	"time"

	"github.com/go-msvc/store"
	"github.com/pkg/errors"
	"github.com/satori/uuid"
)

//Config ...
type Config struct{}

//New ...
func (c Config) New(itemName string, itemType reflect.Type) (store.IStore, error) {
	// if err := c.Validate(); err != nil {
	// 	return nil, errors.Wrapf(err, "invalid config")
	// }
	if err := store.ValidateUserType(itemType); err != nil {
		return nil, errors.Wrapf(err, "cannot store %v", itemType)
	}
	return &memoryStore{
		itemName: itemName,
		itemType: itemType,
		id:       make(map[store.ID][]memItem),
	}, nil
}

type memoryStore struct {
	itemName string
	itemType reflect.Type
	id       map[store.ID][]memItem
}

type memItem struct {
	info store.ItemInfo
	data interface{}
}

func (s memoryStore) Name() string {
	return s.itemName
}

func (s memoryStore) Type() reflect.Type {
	return s.itemType
}

func (s *memoryStore) Add(v interface{}) (info store.ItemInfo, err error) {
	newID := store.ID(uuid.NewV1().String())
	item := memItem{
		info: store.ItemInfo{
			ID:        newID,
			Rev:       1,
			Timestamp: time.Now(),
			UserID:    "",
		}, data: v}

	s.id[newID] = []memItem{item}
	return item.info, nil
}

func (s memoryStore) Get(id store.ID) (interface{}, store.ItemInfo, error) {
	if revs, ok := s.id[id]; ok {
		nrRevs := len(revs)
		lastRev := revs[nrRevs-1]
		return lastRev.data, lastRev.info, nil
	}
	return nil, store.ItemInfo{}, errors.Errorf("id=%s not found", id)
} //memoryStore.Get()

func (s memoryStore) GetInfo(id store.ID) (info store.ItemInfo, err error) {
	if revs, ok := s.id[id]; ok {
		nrRevs := len(revs)
		lastRev := revs[nrRevs-1]
		return lastRev.info, nil
	}
	return store.ItemInfo{}, errors.Errorf("id=%s not found", id)
}

func (s memoryStore) Upd(id store.ID, v interface{}) (info store.ItemInfo, err error) {
	revs, ok := s.id[id]
	if !ok {
		return store.ItemInfo{}, errors.Errorf("id:\"%s\" not found", id)
	}

	nrRevs := len(revs)
	lastRev := revs[nrRevs-1]

	//create new rev
	newItem := lastRev
	newItem.info.Rev = lastRev.info.Rev + 1
	newItem.info.Timestamp = time.Now()
	newItem.data = v
	revs = append(revs, newItem)
	s.id[id] = revs

	return newItem.info, nil
}

func (s memoryStore) Del(id store.ID) error {
	delete(s.id, id)
	return nil
}
