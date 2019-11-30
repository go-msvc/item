package store

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-msvc/errors"
)

//DoStoreTest is called in implementation tests to do
//consistent testing on all store implementations
func DoStoreTest(t *testing.T, c IStoreConfig) {
	s, err := c.New("test", reflect.TypeOf(d{}))
	if err != nil {
		t.Fatalf("failed: %+v", err)
	}

	t0 := time.Now().Truncate(time.Millisecond) //mongo defaults to millisecond resolution
	d1 := d{I: 12345, S: "67890", T: t0}
	info1, err := s.Add(d1)
	if err != nil || info1.Rev != 1 {
		panic(errors.Wrapf(err, "failed to new: rev=%d, err=%v", info1, err))
	}

	d2, info2, err := s.Get(info1.ID)
	if err != nil || info2.Rev != 1 {
		panic(errors.Wrapf(err, "failed to get: info=%+v, err=%v", info2, err))
	}
	if err := d1.Comp(d2.(d)); err != nil {
		panic(errors.Wrapf(err, "new(%+v) != get(%+v)", d1, d2))
	}

	t1 := time.Now().Truncate(time.Millisecond)
	d3 := d{I: 22222, S: "22222", T: t1}
	info3, err := s.Upd(info1.ID, d3)
	if err != nil || info3.ID != info1.ID || info3.Rev != 2 {
		panic(errors.Wrapf(err, "failed to upd: rev=%d, err=%v", info3, err))
	}

	d4, info4, err := s.Get(info1.ID)
	if err != nil || info4.ID != info1.ID || info4.ID != info3.ID || info4.Rev != 2 {
		panic(errors.Wrapf(err, "failed to get: info=%+v, err=%v", info4, err))
	}
	if err := d3.Comp(d4.(d)); err != nil {
		panic(errors.Wrapf(err, "new(%+v) != get(%+v)", d3, d4))
	}

	err = s.Del(info1.ID)
	if err != nil {
		panic(errors.Wrapf(err, "failed to del"))
	}

	//todo: now count must be 0
}

type d struct {
	I int
	S string
	T time.Time
}

func (a d) Comp(b d) error {
	if a.I != b.I {
		return errors.Errorf("i:%d != i:%d", a.I, b.I)
	}
	if a.S != b.S {
		return errors.Errorf("s:%s != s:%s", a.S, b.S)
	}
	if a.T.Sub(b.T) != 0 {
		return errors.Errorf("t:%v != t:%v (diff=%v)", a.T, b.T, a.T.Sub(b.T))
	}
	return nil
}
