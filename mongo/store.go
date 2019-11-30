package mongo

import (
	"github.com/go-msvc/store"
)

func init() {
	store.Register("mongo", config{})
}

type config struct{}

func (c config) New() store.IStore {
	return nil
}
