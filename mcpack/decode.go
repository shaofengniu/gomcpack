package mcpack

import (
	"bytes"
	"errors"
	"reflect"
	"runtime"
)

var (
	errEmptyKey      = errors.New("empty key")
	errUnexpectedEnd = errors.New("unexpected end")
)

func Unmarshal(data []byte, v interface{}) error {
	var d decodeState
	d.init(data)
	return d.unmarshal(v)
}

type Unmarshaler interface {
	UnmarshalMCPACK([]byte) error
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
	if d.savedError != nil {
		return d.savedError
	}
	if d.off != len(d.data) {
		return errUnexpectedEnd
	}
	return nil
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// if it encounters an Unmarshaler, indirect stops and returns that.
// if decodingNull is true, indirect stops at the last pointer so that
// it can be set to nil.
func (d *decodeState) indirect(v reflect.Value, decodingNull bool) (Unmarshaler, reflect.Value) {
	if !v.IsValid() {
		return nil, reflect.Value{}
	}
	// If v is a named type and is addressable
	// start with its address, so that is the type has pointer
	// methods, we find them
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	for {
		// Load value from interface, but only if the result will be
		// usefully addressable
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
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

		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, reflect.Value{}
			}
		}
		v = v.Elem()
	}
	return nil, v
}

func (d *decodeState) value(v reflect.Value) {
	if !v.IsValid() {
		d.next()
		return
	}

	u, pv := d.indirect(v, false)
	if u != nil {
		if err := u.UnmarshalMCPACK(d.next()); err != nil {
			d.error(err)
		}
		return
	}

	v = pv

	switch d.data[d.off] {
	case MCPACKV2_OBJECT:
		d.object(v)
	case MCPACKV2_ARRAY:
		d.array(v)
	case MCPACKV2_STRING:
		d.string(v)
	case MCPACKV2_SHORT_STRING:
		d.shortString(v)
	case MCPACKV2_BINARY:
		d.binary(v)
	case MCPACKV2_SHORT_BINARY:
		d.shortBinary(v)
	case MCPACKV2_INT8:
		d.int8(v)
	case MCPACKV2_INT16:
		d.int16(v)
	case MCPACKV2_INT32:
		d.int32(v)
	case MCPACKV2_INT64:
		d.int64(v)
	case MCPACKV2_UINT8:
		d.uint8(v)
	case MCPACKV2_UINT16:
		d.uint16(v)
	case MCPACKV2_UINT32:
		d.uint32(v)
	case MCPACKV2_UINT64:
		d.uint64(v)
	case MCPACKV2_BOOL:
		d.bool(v)
	case MCPACKV2_FLOAT:
		d.float(v)
	case MCPACKV2_DOUBLE:
		d.double(v)
	case MCPACKV2_NULL:
		d.null(v)
	}
}

func (d *decodeState) next() []byte {
	start := d.off
	typ := d.data[d.off]
	d.off += 1
	klen := int(Int8(d.data[d.off:]))
	d.off += 1
	vlen := 0
	switch typ {
	case MCPACKV2_OBJECT:
		vlen = int(Int32(d.data[d.off:]))
		d.off += 4
	case MCPACKV2_ARRAY:
		vlen = int(Int32(d.data[d.off:]))
		d.off += 4
	case MCPACKV2_STRING:
		vlen = int(Int32(d.data[d.off:]))
		d.off += 4
	case MCPACKV2_SHORT_STRING:
		vlen = int(Int8(d.data[d.off:]))
		d.off += 1
	case MCPACKV2_BINARY:
		vlen = int(Int32(d.data[d.off:]))
		d.off += 4
	case MCPACKV2_SHORT_BINARY:
		vlen = int(Int8(d.data[d.off:]))
		d.off += 1
	case MCPACKV2_INT8:
		vlen = 1
	case MCPACKV2_INT16:
		vlen = 2
	case MCPACKV2_INT32:
		vlen = 4
	case MCPACKV2_INT64:
		vlen = 8
	case MCPACKV2_UINT8:
		vlen = 1
	case MCPACKV2_UINT16:
		vlen = 2
	case MCPACKV2_UINT32:
		vlen = 4
	case MCPACKV2_UINT64:
		vlen = 8
	case MCPACKV2_BOOL:
		vlen = 1
	case MCPACKV2_FLOAT:
		vlen = 4
	case MCPACKV2_DOUBLE:
		vlen = 8
	case MCPACKV2_DATE:
		// FIXME
	case MCPACKV2_NULL:
		vlen = 1
	}
	d.off += klen + vlen
	return d.data[start:d.off]
}

// type(1) | name length(1) | content length (4)
// | raw name bytes | 0x00 | content bytes | 0x00
func (d *decodeState) string(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	d.off += klen // name and 0x00

	val := string(d.data[d.off : d.off+vlen-1])
	d.off += vlen // value and 0x00

	v.SetString(val)
}

func (d *decodeState) stringInterface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	d.off += klen // name and 0x00

	val := string(d.data[d.off : d.off+vlen-1])
	d.off += vlen // value and 0x00

	return val
}

// type(1) | name length(1) | content length(1) | raw name bytes |
// 0x00 | content bytes | 0x00
func (d *decodeState) shortString(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int8(d.data[d.off:]))
	d.off += 1 // content length

	d.off += klen // name and 0x00

	val := string(d.data[d.off : d.off+vlen-1])
	d.off += vlen // value and 0x00

	v.SetString(val)
}

func (d *decodeState) shortStringInterface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int8(d.data[d.off:]))
	d.off += 1 // content length

	d.off += klen // name and 0x00

	val := string(d.data[d.off : d.off+vlen-1])
	d.off += vlen // value and 0x00

	return val
}

// type(1) | name length(1) | content length(4) | raw name bytes |
// 0x00 | content bytes
func (d *decodeState) binary(v reflect.Value) {
	d.off += 1 //type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	d.off += klen // name and 0x00

	val := d.data[d.off : d.off+vlen]
	d.off += vlen // value

	v.SetBytes(val)
}

func (d *decodeState) binaryInterface() interface{} {
	d.off += 1 //type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	d.off += klen // name and 0x00

	val := d.data[d.off : d.off+vlen]
	d.off += vlen // value

	return val
}

// type(1) | name length(1) | content length(1) | raw name bytes |
// 0x00 | content bytes
func (d *decodeState) shortBinary(v reflect.Value) {
	d.off += 1 //type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int8(d.data[d.off:]))
	d.off += 1 // content length

	d.off += klen // name and 0x00

	val := d.data[d.off : d.off+vlen]
	d.off += vlen // value

	v.SetBytes(val)
}

func (d *decodeState) shortBinaryInterface() interface{} {
	d.off += 1 //type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	vlen := int(Int8(d.data[d.off:]))
	d.off += 1 // content length

	d.off += klen // name and 0x00

	val := d.data[d.off : d.off+vlen]
	d.off += vlen // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(1)
func (d *decodeState) int8(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int8(d.data[d.off:])
	d.off += 1 // value

	v.SetInt(int64(val))
}

func (d *decodeState) int8Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int8(d.data[d.off:])
	d.off += 1 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(1)
func (d *decodeState) uint8(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint8(d.data[d.off:])
	d.off += 1 // value

	v.SetUint(uint64(val))
}

func (d *decodeState) uint8Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint8(d.data[d.off:])
	d.off += 1 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(2)
func (d *decodeState) int16(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int16(d.data[d.off:])
	d.off += 2 // value

	v.SetInt(int64(val))
}

func (d *decodeState) int16Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int16(d.data[d.off:])
	d.off += 2 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(2)
func (d *decodeState) uint16(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint16(d.data[d.off:])
	d.off += 2 // value

	v.SetUint(uint64(val))
}

func (d *decodeState) uint16Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint16(d.data[d.off:])
	d.off += 2 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(4)
func (d *decodeState) int32(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int32(d.data[d.off:])
	d.off += 4 // value

	v.SetInt(int64(val))
}

func (d *decodeState) int32Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int32(d.data[d.off:])
	d.off += 4 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(4)
func (d *decodeState) uint32(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint32(d.data[d.off:])
	d.off += 4 // value

	v.SetUint(uint64(val))
}

func (d *decodeState) uint32Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint32(d.data[d.off:])
	d.off += 4 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(8)
func (d *decodeState) int64(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int64(d.data[d.off:])
	d.off += 8 // value

	v.SetInt(val)
}

func (d *decodeState) int64Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Int64(d.data[d.off:])
	d.off += 8 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(8)
func (d *decodeState) uint64(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint64(d.data[d.off:])
	d.off += 8 // value

	v.SetUint(val)
}

func (d *decodeState) uint64Interface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Uint64(d.data[d.off:])
	d.off += 8 // value

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | 0x00
func (d *decodeState) null(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	d.off += 1 // value

	v.Set(reflect.Zero(v.Type()))
}

// type(1) | name length(1) | raw name bytes | 0x00 | 0x00
func (d *decodeState) nullInterface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	d.off += 1 // value

	return nil
}

// type(1) | name length(1) | raw name bytes | 0x00 | 0x00/0x01
func (d *decodeState) bool(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := d.data[d.off]
	d.off += 1

	if val == 0 {
		v.SetBool(false)
	} else {
		v.SetBool(true)
	}
}

func (d *decodeState) boolInterface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := d.data[d.off]
	d.off += 1

	if val == 0 {
		return false
	} else {
		return true
	}
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(4)
func (d *decodeState) float(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Float32(d.data[d.off:])
	d.off += 4

	v.SetFloat(float64(val))
}

func (d *decodeState) floatInterface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Float32(d.data[d.off:])
	d.off += 4

	return val
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes(8)
func (d *decodeState) double(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Float64(d.data[d.off:])
	d.off += 8

	v.SetFloat(val)
}

func (d *decodeState) doubleInterface() interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	d.off += klen

	val := Float64(d.data[d.off:])
	d.off += 8

	return val
}

func (d *decodeState) valueInterface() interface{} {
	switch d.data[d.off] {
	case MCPACKV2_OBJECT:
		return d.objectInterface()
	case MCPACKV2_ARRAY:
		return d.arrayInterface()
	case MCPACKV2_STRING:
		return d.stringInterface()
	case MCPACKV2_SHORT_STRING:
		return d.shortStringInterface()
	case MCPACKV2_BINARY:
		return d.binaryInterface()
	case MCPACKV2_SHORT_BINARY:
		return d.shortBinaryInterface()
	case MCPACKV2_INT8:
		return d.int8Interface()
	case MCPACKV2_INT16:
		return d.int16Interface()
	case MCPACKV2_INT32:
		return d.int32Interface()
	case MCPACKV2_INT64:
		return d.int64Interface()
	case MCPACKV2_UINT8:
		return d.uint8Interface()
	case MCPACKV2_UINT16:
		return d.uint16Interface()
	case MCPACKV2_UINT32:
		return d.uint32Interface()
	case MCPACKV2_UINT64:
		return d.uint64Interface()
	case MCPACKV2_BOOL:
		return d.boolInterface()
	case MCPACKV2_FLOAT:
		return d.floatInterface()
	case MCPACKV2_DOUBLE:
		return d.doubleInterface()
	case MCPACKV2_NULL:
		return d.nullInterface()
	}
	return nil
}

// type(1) | name length(1) | item size(4) | raw name bytes | 0x00
// | members number(4) | member1 | ... | memberN
func (d *decodeState) object(v reflect.Value) {
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
		v.Set(reflect.ValueOf(d.objectInterface()))
		return
	}

	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	// vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	d.off += klen // name and 0x00

	n := int(Int32(d.data[d.off:]))
	d.off += 4 // member number

	var mapElem reflect.Value
	for i := 0; i < n; i++ {
		subk := d.key()
		var subv reflect.Value

		if v.Kind() == reflect.Map {
			elemType := v.Type().Elem()
			if !mapElem.IsValid() {
				mapElem = reflect.New(elemType).Elem()
			} else {
				mapElem.Set(reflect.Zero(elemType))
			}
			subv = mapElem
		} else {
			var f *field
			fields := cachedTypeFields(v.Type())
			for i := range fields {
				ff := &fields[i]
				if bytes.Equal(ff.nameBytes, subk) {
					f = ff
					break
				}
				if f == nil && ff.equalFold(ff.nameBytes, subk) {
					f = ff
				}
			}
			if f != nil {
				subv = v
				for _, i := range f.index {
					if v.Kind() == reflect.Ptr {
						if subv.IsNil() {
							subv.Set(reflect.New(subv.Type()).Elem())
						}
						subv = subv.Elem()
					}
					subv = subv.Field(i)
				}
			}
		}

		d.value(subv)

		// Write value back to map
		if v.Kind() == reflect.Map {
			kv := reflect.ValueOf(subk).Convert(v.Type().Key())
			v.SetMapIndex(kv, subv)
		}
	}
}

func (d *decodeState) objectInterface() map[string]interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	// vlen := int(Int32(d.data[d.off:]))
	d.off += 4 // content length

	d.off += klen // name and 0x00

	n := int(Int32(d.data[d.off:]))
	d.off += 4 // member number

	m := make(map[string]interface{})
	for i := 0; i < n; i++ {
		subk := d.key()
		m[string(subk)] = d.valueInterface()
	}

	return m
}

//FIXME: fix when v is invalid
// type(1) | name length(1) | item size(4) | raw name bytes | 0x00
// | element number(4) | element1 | ... | elementN
func (d *decodeState) array(v reflect.Value) {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	// vlen := int(Int32(d.data[d.off:]))
	d.off += 4 //  content length

	//var key string
	d.off += klen

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
			d.value(reflect.Value{})
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

func (d *decodeState) arrayInterface() []interface{} {
	d.off += 1 // type

	klen := int(Int8(d.data[d.off:]))
	d.off += 1 // name length

	// vlen := int(Int32(d.data[d.off:]))
	d.off += 4 //  content length

	//var key string
	d.off += klen

	n := int(Int32(d.data[d.off:]))
	d.off += 4 // member number

	v := make([]interface{}, n)
	for i := 0; i < n; i++ {
		v[i] = d.valueInterface()
	}
	return v
}

func (d *decodeState) key() []byte {
	var kstart int
	switch d.data[d.off] {
	case MCPACKV2_INT8, MCPACKV2_INT16, MCPACKV2_INT32, MCPACKV2_INT64,
		MCPACKV2_UINT8, MCPACKV2_UINT16, MCPACKV2_UINT32, MCPACKV2_UINT64,
		MCPACKV2_BOOL, MCPACKV2_FLOAT, MCPACKV2_DOUBLE, MCPACKV2_NULL:
		kstart = 2 // type + klen
	case MCPACKV2_SHORT_BINARY, MCPACKV2_SHORT_STRING:
		kstart = 3 // type + klen + vlen(1)
	case MCPACKV2_BINARY, MCPACKV2_STRING, MCPACKV2_OBJECT, MCPACKV2_ARRAY:
		kstart = 6 // type + klen + vlen(4)
	}
	klen := int(Int8(d.data[d.off+1:]))
	if klen <= 0 {
		d.error(errEmptyKey)
	}
	return d.data[d.off+kstart : d.off+kstart+klen-1]
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
