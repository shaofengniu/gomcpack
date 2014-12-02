package nf

import (
	"io/ioutil"
	"strings"
	"testing"
)

func test(w ResponseWriter, r *Request) {
	content, err := ioutil.ReadAll(r.Body)
	println("recv", string(content))
	if err != nil {
		w.Write([]byte("fail"))
		return
	}
	w.Write([]byte("pong"))
}

func TestServer(t *testing.T) {
	s := &Server{
		Addr:    ":8888",
		Handler: HandlerFunc(test),
	}
	go s.ListenAndServe()

	c, err := Dial("localhost:8888")
	if err != nil {
		t.Error(err)
	}
	req, err := NewRequest(strings.NewReader("ping"))
	if err != nil {
		t.Error(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Error(err)
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if string(content) != "pong" {
		t.Errorf("got %s", string(content))
	}
}
