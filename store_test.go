package store_test

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

type d struct {
	I int
	S string
	T time.Time
}

func Test1(t *testing.T) {
	//make type like D but include _id for mongo
	fields := make([]reflect.StructField, 0)
	nf := reflect.StructField{
		Name:      "ID",
		PkgPath:   "", //make it exported,
		Type:      reflect.TypeOf(""),
		Tag:       reflect.StructTag(""),
		Offset:    0,
		Index:     []int{},
		Anonymous: false,
	}
	fields = append(fields, nf)

	dt := reflect.TypeOf(d{})
	for i := 0; i < dt.NumField(); i++ {
		f := dt.Field(i)
		nf := reflect.StructField{
			Name:      strings.ToUpper(f.Name[0:1]) + f.Name[1:],
			PkgPath:   "", //make it exported,
			Type:      f.Type,
			Tag:       f.Tag,
			Offset:    f.Offset,
			Index:     f.Index,
			Anonymous: false,
		}
		fields = append(fields, nf)
	}
	st := reflect.StructOf(fields)
	d := reflect.New(st).Interface()
	t.Logf("T=%T, v=%+v", d, d)
}
