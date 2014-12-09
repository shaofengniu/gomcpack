package mcpacknpc

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"

	"gitlab.baidu.com/niushaofeng/gomcpack/mcpack"
	"gitlab.baidu.com/niushaofeng/gomcpack/npc"
)

// Precompute the reflect type for error.  Can't use error directly
// because Typeof takes an empty interface value.  This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type Handler struct {
	Fn        reflect.Value
	ArgType   reflect.Type
	ReplyType reflect.Type

	sync.Mutex
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

func NewHandler(fn interface{}) (*Handler, error) {
	ftype := reflect.TypeOf(fn)
	if ftype.NumIn() != 2 {
		return nil, fmt.Errorf("function %s has wrong number of ins: %d", ftype.Name(), ftype.NumIn())
	}
	argType := ftype.In(0)
	if !isExportedOrBuiltinType(argType) {
		return nil, fmt.Errorf("function %s argument type not exported: %v", ftype.Name(), argType)
	}
	replyType := ftype.In(1)
	if replyType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("function %s reply type not a pointer: %v", ftype.Name(), replyType)
	}
	if !isExportedOrBuiltinType(replyType) {
		return nil, fmt.Errorf("function %s reply type not exported: %v", ftype.Name(), replyType)
	}
	if ftype.NumOut() != 1 {
		return nil, fmt.Errorf("function %s has wrong number of outs: %v", ftype.Name(), ftype.NumOut())
	}
	if returnType := ftype.Out(0); returnType != typeOfError {
		return nil, fmt.Errorf("function %s returns %s not error", ftype.Name(), returnType.String())
	}
	return &Handler{Fn: reflect.ValueOf(fn), ArgType: argType, ReplyType: replyType}, nil
}

func (h *Handler) Serve(w npc.ResponseWriter, r *npc.Request) {
	argIsValue := false
	var argv, replyv reflect.Value
	if h.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(h.ArgType.Elem())
	} else {
		argv = reflect.New(h.ArgType)
		argIsValue = true
	}
	if err := h.readRequest(r, argv.Interface()); err != nil {
		log.Printf("readRequest: %v", err)
		return
	}
	if argIsValue {
		argv = argv.Elem()
	}
	replyv = reflect.New(h.ReplyType.Elem())

	returnValues := h.Fn.Call([]reflect.Value{argv, replyv})
	errInter := returnValues[0].Interface()
	if errInter != nil {
		log.Printf("function call: %v", errInter.(error).Error())
		return
	}
	if err := h.sendResponse(w, replyv.Interface()); err != nil {
		log.Printf("sendResponse: %v", err)
		return
	}
}

func (h *Handler) readRequest(r *npc.Request, arg interface{}) error {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return mcpack.Unmarshal(content, arg)
}

func (h *Handler) sendResponse(w npc.ResponseWriter, reply interface{}) error {
	content, err := mcpack.Marshal(reply)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}
