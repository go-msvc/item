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
		id:       make(map[store.ID][]item),
	}, nil
}

type memoryStore struct {
	itemName string
	itemType reflect.Type
	id       map[store.ID][]item
}

type item struct {
	rev    int
	ts     time.Time
	userID store.ID
	data   interface{}
}

func (s memoryStore) Name() string {
	return s.itemName
}

func (s *memoryStore) Add(v interface{}) (id store.ID, rev int, err error) {
	newID := store.ID(uuid.NewV1().String())
	s.id[newID] = []item{item{rev: 1, ts: time.Now(), userID: "", data: v}}
	return newID, 1, nil
}

func (s memoryStore) Get(id store.ID) (interface{}, store.ItemInfo, error) {
	if revs, ok := s.id[id]; ok {
		nrRevs := len(revs)
		lastRevItem := revs[nrRevs-1]
		return lastRevItem.data, store.ItemInfo{
			ID:        id,
			Rev:       lastRevItem.rev,
			Timestamp: lastRevItem.ts,
			UserID:    lastRevItem.userID,
		}, nil
	}
	return nil, store.ItemInfo{}, errors.Errorf("id=%s not found", id)
} //memoryStore.Get()

func (s memoryStore) GetInfo(id store.ID) (info store.ItemInfo, err error) {
	if revs, ok := s.id[id]; ok {
		nrRevs := len(revs)
		lastRevItem := revs[nrRevs-1]
		return store.ItemInfo{
			ID:        id,
			Rev:       lastRevItem.rev,
			Timestamp: lastRevItem.ts,
			UserID:    lastRevItem.userID,
		}, nil
	}
	return store.ItemInfo{}, errors.Errorf("id=%s not found", id)
}

func (s memoryStore) Upd(id store.ID, v interface{}) (rev int, err error) {
	revs, ok := s.id[id]
	if !ok {
		return 0, errors.Errorf("id:\"%s\" not found", id)
	}

	nrRevs := len(revs)
	last := revs[nrRevs-1]

	//create new rev
	newItem := last
	newItem.rev = last.rev + 1
	newItem.ts = time.Now()
	newItem.data = v
	revs = append(revs, newItem)
	s.id[id] = revs

	return newItem.rev, nil
}

func (s memoryStore) Del(id store.ID) error {
	delete(s.id, id)
	return nil
}
