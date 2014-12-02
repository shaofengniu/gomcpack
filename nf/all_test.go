package nf

import (
	"log"
	"testing"
)

func add(ResponseWriter w, Request *r) {

}

type req struct {
	A int
	B int
}

type resp struct {
	C int
}

func ServerTest(t *testing.T) {
	s := &Server{
		Addr:    ":8888",
		Handler: add,
	}
	go log.Fatal(s.ListenAndServe())

}
