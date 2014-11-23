package gomcpack

import (
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode"
)

func Marshal(v interface{}) ([]byte, error) {
	e := &encodeState{}
	err := e.marshal(v)
	if err != nil {
		return nil, err
	}
	return e.data[:e.off], nil
}

type encodeState struct {
	data    []byte
	off     int
	scratch [64]byte
}

func max(l, r int) int {
	if l >= r {
		return l
	} else {
		return r
	}
}

func (e *encodeState) resizeIfNeeded(n int) {
	if e.off+n >= cap(e.data) {
		newcap := max(cap(e.data)*2, e.off+n)
		newdata := make([]byte, newcap, newcap)
		copy(newdata, e.data)
		e.data = newdata
	}
}

func (e *encodeState) marshal(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if s, ok := r.(string); ok {
				panic(s)
			}
			err = r.(error)
		}
	}()
	e.reflectValue("", reflect.ValueOf(v))
	return nil
}

func (e *encodeState) reflectValue(k string, v reflect.Value) {
	valueEncoder(v)(e, k, v)
}

type encoderFunc func(e *encodeState, k string, v reflect.Value)

var encoderCache struct {
	sync.RWMutex
	m map[reflect.Type]encoderFunc
}

func valueEncoder(v reflect.Value) encoderFunc {
	if !v.IsValid() {
		return invalidValueEncoder
	}
	return typeEncoder(v.Type())
}

func typeEncoder(t reflect.Type) encoderFunc {
	encoderCache.RLock()
	f := encoderCache.m[t]
	encoderCache.RUnlock()
	if f != nil {
		return f
	}

	encoderCache.Lock()
	if encoderCache.m == nil {
		encoderCache.m = make(map[reflect.Type]encoderFunc)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	encoderCache.m[t] = func(e *encodeState, k string, v reflect.Value) {
		wg.Wait()
		f(e, k, v)
	}
	encoderCache.Unlock()

	f = newTypeEncoder(t, true)
	wg.Done()
	encoderCache.Lock()
	encoderCache.m[t] = f
	encoderCache.Unlock()
	return f
}

func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
	switch t.Kind() {
	case reflect.Bool:
		return boolEncoder
	case reflect.Int:
		return int64Encoder
	case reflect.Int8:
		return int8Encoder
	case reflect.Int16:
		return int16Encoder
	case reflect.Int32:
		return int32Encoder
	case reflect.Int64:
		return int64Encoder
	case reflect.Uint:
		return uint64Encoder
	case reflect.Uint8:
		return uint8Encoder
	case reflect.Uint16:
		return uint16Encoder
	case reflect.Uint32:
		return uint32Encoder
	case reflect.Uint64:
		return uint64Encoder
	case reflect.Uintptr:
		return uint64Encoder
	case reflect.Float32:
		return float32Encoder
	case reflect.Float64:
		return float64Encoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Struct:
		return newStructEncoder(t)
	case reflect.Map:
		return newMapEncoder(t)
	case reflect.Slice:
		return newSliceEncoder(t)
	case reflect.Array:
		return newArrayEncoder(t)
	case reflect.Ptr:
		return newPtrEncoder(t)
	default:
		return unsupportedTypeEncoder
	}
}

func unsupportedTypeEncoder(e *encodeState, k string, v reflect.Value) {
}

func invalidValueEncoder(e *encodeState, k string, v reflect.Value) {
}

func nilEncoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 1)
	e.data[e.off] = MCPACKV2_NULL
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	e.data[e.off] = 0
	e.off++
}

func boolEncoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 1)
	e.data[e.off] = MCPACKV2_BOOL
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	if v.Bool() {
		e.data[e.off] = 1
	} else {
		e.data[e.off] = 0
	}
	e.off++
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes
func int8Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_INT8
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutInt8(e.data[e.off:], int8(v.Int()))
	e.off += 1
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes
func int16Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_INT16
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutInt16(e.data[e.off:], int16(v.Int()))
	e.off += 2
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes
func int32Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_INT32
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutInt32(e.data[e.off:], int32(v.Int()))
	e.off += 4
}

// type(1) | name length(1) | raw name bytes | 0x00 | value bytes
func int64Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_INT64
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutInt64(e.data[e.off:], v.Int())
	e.off += 8
}

func uint8Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_UINT8
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutUint8(e.data[e.off:], uint8(v.Uint()))
	e.off += 1
}
func uint16Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_UINT16
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutUint16(e.data[e.off:], uint16(v.Uint()))
	e.off += 2
}
func uint32Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_UINT32
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutUint32(e.data[e.off:], uint32(v.Uint()))
	e.off += 4
}
func uint64Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_UINT64
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutUint64(e.data[e.off:], uint64(v.Uint()))
	e.off += 8
}

func float32Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 4)
	e.data[e.off] = MCPACKV2_FLOAT
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutFloat32(e.data[e.off:], float32(v.Float()))
	e.off += 4
}

func float64Encoder(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + len(k) + 1 + 8)
	e.data[e.off] = MCPACKV2_DOUBLE
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	PutFloat64(e.data[e.off:], v.Float())
	e.off += 8
}

func stringEncoder(e *encodeState, k string, v reflect.Value) {
	//type(1) | klen(1) | vlen(4) | key(len(k)) | 0x00 | value | 0x00
	e.resizeIfNeeded(1 + 1 + 4 + len(k) + 1 + v.Len() + 1)
	e.data[e.off] = MCPACKV2_STRING
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	vlenpos := e.off // content length pos
	e.off += 4

	if len(k) > 0 { // key and 0x00
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	vpos := e.off
	e.off += copy(e.data[e.off:], v.String())
	e.data[e.off] = 0 // value and 0x00
	e.off++

	PutInt32(e.data[vlenpos:], int32(e.off-vpos))
}

func binaryEncoder(e *encodeState, k string, v reflect.Value) {
	//type(1) | klen(1) | vlen(4) | key(len(k)) | 0x00 | value
	e.resizeIfNeeded(1 + 1 + 4 + len(k) + 1 + v.Len())
	e.data[e.off] = MCPACKV2_BINARY
	e.off++

	if len(k) > 0 { // key len
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	vlenpos := e.off
	e.off += 4 // content length

	if len(k) > 0 { // key and 0x00
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}
	vpos := e.off
	e.off += copy(e.data[e.off:], v.Bytes()) //value

	PutInt32(e.data[vlenpos:], int32(e.off-vpos))
}

func interfaceEncoder(e *encodeState, k string, v reflect.Value) {
	if v.IsNil() {
		nilEncoder(e, k, v)
		return
	}
	e.reflectValue(k, v.Elem())
}

type structEncoder struct {
	fields    []field
	fieldEncs []encoderFunc
}

func (se *structEncoder) encode(e *encodeState, k string, v reflect.Value) {
	// type(1) | klen(1) | vlen(4) | key(len(k)) | 0x00 | field number(4)
	e.resizeIfNeeded(1 + 1 + 4 + len(k) + 1 + 4)

	e.data[e.off] = MCPACKV2_OBJECT
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	vlenpos := e.off
	e.off += 4

	if len(k) > 0 {
		n := copy(e.data[e.off:], k)
		e.off += n
		e.data[e.off] = 0
		e.off++
	}

	vpos := e.off
	PutInt32(e.data[e.off:], int32(len(se.fields)))
	e.off += 4

	for i, f := range se.fields {
		fv := fieldByIndex(v, f.index)
		if !fv.IsValid() || f.omitEmpty && isEmptyValue(fv) {
			continue
		}
		se.fieldEncs[i](e, f.name, fv)
	}
	PutInt32(e.data[vlenpos:], int32(e.off-vpos))
}

func newStructEncoder(t reflect.Type) encoderFunc {
	fields := cachedTypeFields(t)
	se := &structEncoder{
		fields:    fields,
		fieldEncs: make([]encoderFunc, len(fields)),
	}
	for i, f := range fields {
		se.fieldEncs[i] = typeEncoder(typeByIndex(t, f.index))
	}
	return se.encode
}

type mapEncoder struct {
	elemEnc encoderFunc
}

func (me *mapEncoder) encode(e *encodeState, k string, v reflect.Value) {
	e.resizeIfNeeded(1 + 1 + 4 + len(k) + 1 + 4)

	e.data[e.off] = MCPACKV2_OBJECT
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	vlenpos := e.off
	e.off += 4

	if len(k) > 0 {
		n := copy(e.data[e.off:], k)
		e.off += n
		e.data[e.off] = 0
		e.off++
	}

	vpos := e.off
	PutInt32(e.data[e.off:], int32(len(v.MapKeys())))
	e.off += 4

	for _, k := range v.MapKeys() {
		me.elemEnc(e, k.String(), v.MapIndex(k))
	}
	PutInt32(e.data[vlenpos:], int32(e.off-vpos))
}

func newMapEncoder(t reflect.Type) encoderFunc {
	if t.Key().Kind() != reflect.String {
		return unsupportedTypeEncoder
	}
	me := &mapEncoder{typeEncoder(t.Elem())}
	return me.encode
}

type sliceEncoder struct {
	arrayEnc encoderFunc
}

func (se *sliceEncoder) encode(e *encodeState, k string, v reflect.Value) {
	if v.IsNil() {
		nilEncoder(e, k, v)
		return
	}
	se.arrayEnc(e, k, v)
}

func newSliceEncoder(t reflect.Type) encoderFunc {
	if t.Elem().Kind() == reflect.Uint8 {
		return binaryEncoder
	}
	enc := &sliceEncoder{newArrayEncoder(t)}
	return enc.encode
}

type arrayEncoder struct {
	elemEnc encoderFunc
}

func (ae *arrayEncoder) encode(e *encodeState, k string, v reflect.Value) {
	// type(1) | klen(1) | vlen(4) | key(len(0)) | 0x00 | field
	// number(4)
	e.resizeIfNeeded(1 + 1 + 4 + len(k) + 1 + 4)

	e.data[e.off] = MCPACKV2_ARRAY
	e.off++

	if len(k) > 0 {
		e.data[e.off] = byte(len(k) + 1)
	} else {
		e.data[e.off] = 0
	}
	e.off++

	vlenpos := e.off
	e.off += 4

	if len(k) > 0 {
		e.off += copy(e.data[e.off:], k)
		e.data[e.off] = 0
		e.off++
	}

	vpos := e.off
	PutInt32(e.data[e.off:], int32(v.Len()))
	e.off += 4

	for i := 0; i < v.Len(); i++ {
		ae.elemEnc(e, "", v.Index(i))
	}

	PutInt32(e.data[vlenpos:], int32(e.off-vpos))
}

func newArrayEncoder(t reflect.Type) encoderFunc {
	enc := &arrayEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type ptrEncoder struct {
	elemEnc encoderFunc
}

func (pe *ptrEncoder) encode(e *encodeState, k string, v reflect.Value) {
	if v.IsNil() {
		nilEncoder(e, k, v)
		return
	}
	pe.elemEnc(e, k, v.Elem())
}

func newPtrEncoder(t reflect.Type) encoderFunc {
	enc := &ptrEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type field struct {
	name      string
	nameBytes []byte
	equalFold func(s, t []byte) bool

	tag       bool
	index     []int
	typ       reflect.Type
	omitEmpty bool
}

func fillField(f field) field {
	f.nameBytes = []byte(f.name)
	f.equalFold = foldFunc(f.nameBytes)
	return f
}

var fieldCache struct {
	sync.RWMutex
	m map[reflect.Type][]field
}

func cachedTypeFields(t reflect.Type) []field {
	fieldCache.RLock()
	f := fieldCache.m[t]
	fieldCache.RUnlock()
	if f != nil {
		return f
	}

	f = typeFields(t)
	if f == nil {
		f = []field{}
	}

	fieldCache.Lock()
	if fieldCache.m == nil {
		fieldCache.m = map[reflect.Type][]field{}
	}
	fieldCache.m[t] = f
	fieldCache.Unlock()
	return f
}

func typeFields(t reflect.Type) []field {
	current := []field{}
	next := []field{{typ: t}}

	count := map[reflect.Type]int{}
	nextCount := map[reflect.Type]int{}

	visited := map[reflect.Type]bool{}

	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				if sf.PkgPath != "" {
					continue
				}
				tag := sf.Tag.Get("json")
				if tag == "-" {
					continue
				}
				name, opts := parseTag(tag)
				if !isValidTag(name) {
					name = ""
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i
				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Ptr {
					ft = ft.Elem() // FIXME: why???
				}

				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					fields = append(fields, fillField(field{
						name:      name,
						tag:       tagged,
						index:     index,
						typ:       ft,
						omitEmpty: opts.Contains("omitempty"),
					}))
					if count[f.typ] > 1 {
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, fillField(field{name: ft.Name(), index: index, typ: ft}))
				}

			}
		}

	}
	sort.Sort(byName(fields))
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		fi := fields[i]
		name := fi.name
		for advance = 1; i+advance < len(fields); advance++ {
			fj := fields[i+advance]
			if fj.name != name {
				break
			}
		}
		if advance == 1 {
			out = append(out, fi)
			continue
		}
		dominant, ok := dominantField(fields[i : i+advance])
		if ok {
			out = append(out, dominant)
		}
	}

	fields = out
	sort.Sort(byIndex(fields))

	return fields
}

func dominantField(fields []field) (field, bool) {
	// The fields are sorted in increasing index-length order. The winner
	// must therefore be one with the shortest index length. Drop all
	// longer entries, which is easy: just truncate the slice.
	length := len(fields[0].index)
	tagged := -1 // Index of first tagged field.
	for i, f := range fields {
		if len(f.index) > length {
			fields = fields[:i]
			break
		}
		if f.tag {
			if tagged >= 0 {
				// Multiple tagged fields at the same level: conflict.
				// Return no field.
				return field{}, false
			}
			tagged = i
		}
	}
	if tagged >= 0 {
		return fields[tagged], true
	}
	// All remaining fields have the same length. If there's more than one,
	// we have a conflict (two fields named "X" at the same level) and we
	// return no field.
	if len(fields) > 1 {
		return field{}, false
	}
	return fields[0], true
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, i := range index {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		t = t.Field(i).Type
	}
	return t
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		default:
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
				return false
			}
		}
	}
	return true
}

// byName sorts field by name, breaking ties with depth,
// then breaking ties with "name came from json tag", then
// breaking ties with index sequence.
type byName []field

func (x byName) Len() int { return len(x) }

func (x byName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byName) Less(i, j int) bool {
	if x[i].name != x[j].name {
		return x[i].name < x[j].name
	}
	if len(x[i].index) != len(x[j].index) {
		return len(x[i].index) < len(x[j].index)
	}
	if x[i].tag != x[j].tag {
		return x[i].tag
	}
	return byIndex(x).Less(i, j)
}

// byIndex sorts field by index sequence.
type byIndex []field

func (x byIndex) Len() int { return len(x) }

func (x byIndex) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byIndex) Less(i, j int) bool {
	for k, xik := range x[i].index {
		if k >= len(x[j].index) {
			return false
		}
		if xik != x[j].index[k] {
			return xik < x[j].index[k]
		}
	}
	return len(x[i].index) < len(x[j].index)
}
