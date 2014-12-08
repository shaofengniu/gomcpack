package nf_test

import (
	"io/ioutil"
	"strings"
	"testing"

	. "gitlab.baidu.com/niushaofeng/gomcpack/nf"
	"gitlab.baidu.com/niushaofeng/gomcpack/nf/nftest"
)

func TestServer(t *testing.T) {
	defer afterTest(t)
	s := nftest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		content, err := ioutil.ReadAll(r.Body)
		if string(content) != "ping" {
			t.Errorf("expected ping, got %s", string(content))
		}
		if err != nil {
			t.Error(err)
		}
		w.Write([]byte("pong"))
	}))
	defer s.Close()

	for i := 0; i < 10; i++ {
		c, err := Dial(s.Listener.Addr().String())
		if err != nil {
			t.Error(err)
		}
		for j := 0; j < 10; j++ {
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
				t.Errorf("expected pong, got %s", string(content))
			}
		}
		c.Close()
	}
}

func TestServerTimeouts(t *testing.T) {
	defer afterTest(t)
	reqNum := 0
	ts := nftest.NewUnstartedServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		reqNum++
		fmt.Fprint(res, "req=%d", reqNum)
	}))

}
