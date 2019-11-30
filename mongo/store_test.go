package mongo_test

import (
	"testing"

	"github.com/go-msvc/store"
	"github.com/go-msvc/store/mongo"
)

func Test1(t *testing.T) {
	store.DoStoreTest(t, mongo.Config{
		Database: "test",
	})
}
