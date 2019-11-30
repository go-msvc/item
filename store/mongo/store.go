package mongo

import (
	"github.com/go-msvc/item"
)

func init() {
	item.RegisterStore("mongo", config{})
}

type config struct{}

func (c config) New() item.IStore {
	return nil
}
