package gomcpack

import (
	"fmt"
	"reflect"
	"runtime"
)

const (
	MCPACKV2_INVALID      = 0x00
	MCPACKV2_OBJECT       = 0x10
	MCPACKV2_ARRAY        = 0x20
	MCPACKV2_STRING       = 0x50
	MCPACKV2_RAW          = 0x60
	MCPACKV2_INT8         = 0x11
	MCPACKV2_INT16        = 0x12
	MCPACKV2_INT32        = 0x14
	MCPACKV2_INT64        = 0x18
	MCPACKV2_UINT8        = 0x21
	MCPACKV2_UINT16       = 0x22
	MCPACKV2_UINT32       = 0x24
	MCPACKV2_UINT64       = 0x28
	MCPACKV2_BOOL         = 0x31
	MCPACKV2_FLOAT        = 0x44
	MCPACKV2_DOUBLE       = 0x48
	MCPACKV2_DATE         = 0x58
	MCPACKV2_NULL         = 0x61
	MCPACKV2_SHORT_ITEM   = 0x80
	MCPACKV2_FIXED_ITEM   = 0xf0
	MCPACKV2_DELETED_ITEM = 0x70

	MCPACKV2_SHORT_STRING = MCPACKV2_STRING | MCPACKV2_SHORT_ITEM
)

func Unmarshal(data []byte, v interface{}) error {
	var d decodeState
	d.init(data)
	return d.unmarshal(v)
}

type decodeState struct {
	data       []byte
	off        int
	savedError error
	tempstr    string
}

func (d *decodeState) init(data []byte) *decodeState {
	d.data = data
	d.off = 0
	d.savedError = nil
	return d
}

func (d *decodeState) error(err error) {
	panic(err)
}

func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = err
	}
}

func (d *decodeState) unmarshal(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	d.value(rv)
	return d.savedError
}

func (d *decodeState) indirect(v reflect.Value, decodingNull bool) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (decodingNull || e.Elem().Kind() == reflect.Ptr) {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && decodingNull && v.CanSet() {
			break
		}

		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		v = v.Elem()
	}
	return v
}

func (d *decodeState) value(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch d.data[d.off] {
	case MCPACKV2_OBJECT:
		d.object(v)
	case MCPACKV2_ARRAY:
		d.array(v)
	case MCPACKV2_STRING:
		d.string(v)
	case MCPACKV2_SHORT_STRING:
	case MCPACKV2_RAW:
	case MCPACKV2_INT8:
	case MCPACKV2_INT16:
	case MCPACKV2_INT32:
	case MCPACKV2_INT64:
	case MCPACKV2_UINT8:
	case MCPACKV2_UINT16:
	case MCPACKV2_UINT32:
	case MCPACKV2_UINT64:
	case MCPACKV2_BOOL:
	case MCPACKV2_FLOAT:
	case MCPACKV2_DOUBLE:
	case MCPACKV2_DATE:
	case MCPACKV2_NULL:
	}
}

// type(1) | name length(1) | content length (4)
// | raw name bytes | 0x00 | content bytes | 0x00
func (d *decodeState) parseString() (string, string) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	var key string
	if klen > 0 {
		key = string(d.data[d.off : d.off+klen-1])
		d.off += klen // name and 0x00
	}

	val := string(d.data[d.off : d.off+vlen])
	d.off += vlen + 1 // value and 0x00

	return key, val
}

func (d *decodeState) string(v reflect.Value) {
	v = d.indirect(v, false)
	_, val := d.parseString()
	v.SetString(val)
}

// type(1) | name length(1) | item size(4) | raw name bytes | 0x00
// | members number(4) | member1 | ... | memberN
func (d *decodeState) object(v reflect.Value) {
	v = d.indirect(v, false)

	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	// vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	// FIXME
	//	var key string
	if klen > 0 {
		//		key = string(d.data[d.off : d.off+klen-1])
		d.off += klen // name and 0x00
	}

	n := int(Int32(d.data[d.off:]))
	d.off += 4 // member number

	for i := 0; i < n; i++ {
		switch d.data[d.off] {
		case MCPACKV2_OBJECT:
		case MCPACKV2_ARRAY:
		case MCPACKV2_STRING:
			key, val := d.parseString()
			field := fieldByTag(v, key)
			field.SetString(val)
		case MCPACKV2_SHORT_STRING:
		case MCPACKV2_RAW:
		case MCPACKV2_INT8:
		case MCPACKV2_INT16:
		case MCPACKV2_INT32:
		case MCPACKV2_INT64:
		case MCPACKV2_UINT8:
		case MCPACKV2_UINT16:
		case MCPACKV2_UINT32:
		case MCPACKV2_UINT64:
		case MCPACKV2_BOOL:
		case MCPACKV2_FLOAT:
		case MCPACKV2_DOUBLE:
		case MCPACKV2_DATE:
		case MCPACKV2_NULL:
		}

	}
}

// type(1) | name length(1) | item size(4) | raw name bytes | 0x00
// | element number(4) | element1 | ... | elementN
func (d *decodeState) array(v reflect.Value) {
	v = d.indirect(v, false)

	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	// vlen := int(Int32(d.data[d.off:]))
	d.off += 4 //  content length

	//var key string
	if klen > 0 {
		//	key = string(d.data[d.off : d.off+klen-1])
		d.off += klen
	}

	n := int(Int32(d.data[d.off:]))
	d.off += 4 // member number

	if v.Kind() == reflect.Slice {
		if n > v.Cap() {
			newv := reflect.MakeSlice(v.Type(), n, n)
			v.Set(newv)
		}
		v.SetLen(n)
	}

	for i := 0; i < n; i++ {
		if i < v.Len() {
			d.value(v.Index(i))
		} else {
			// Ran out of fixed array: skip
		}
	}

	if n < v.Len() {
		if v.Kind() == reflect.Array {
			z := reflect.Zero(v.Type().Elem())
			for i := 0; i < v.Len(); i++ {
				v.Index(i).Set(z)
			}
		}
	}

	if n == 0 && v.Kind() == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}
}

func fieldByTag(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if t.Field(i).Tag.Get("json") == name {
			return v.Field(i)
		}
	}
	panic(fmt.Errorf("field %s not found", name))
}

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "mcpack: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "mcpack: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "mcpack: Unmarshal(nil " + e.Type.String() + ")"
}
