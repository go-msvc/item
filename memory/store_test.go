package memory_test

import (
	"testing"

	"github.com/go-msvc/store"
	"github.com/go-msvc/store/memory"
)

func Test1(t *testing.T) {
	store.DoStoreTest(t, memory.Config{})
}
