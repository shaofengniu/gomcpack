package nf_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	. "gitlab.baidu.com/niushaofeng/gomcpack/nf"
	"gitlab.baidu.com/niushaofeng/gomcpack/nf/nftest"
)

func TestServerPingPong(t *testing.T) {
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
		conn, err := net.Dial("tcp", s.Listener.Addr().String())
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		for j := 0; j < 10; j++ {
			req := NewRequest(strings.NewReader("ping"))
			if _, err := req.Write(conn); err != nil {
				t.Fatalf("Write: %v", err)
			}
			resp, err := ReadResponse(conn)
			if err != nil {
				t.Fatalf("ReadResponse: %v", err)
			}
			content, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Error(err)
			}
			if string(content) != "pong" {
				t.Errorf("expected pong, got %s", string(content))
			}
		}
		conn.Close()
	}
}

/*
func TestServerTimeouts(t *testing.T) {
	defer afterTest(t)
	reqNum := 0
	ts := nftest.NewUnstartedServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		reqNum++
		fmt.Fprintf(w, "req=%d", reqNum)
	}))
	ts.Config.ReadTimeout = 250 * time.Millisecond
	ts.Config.WriteTimeout = 250 * time.Millisecond
	ts.Start()
	defer ts.Close()

	c := NewClient([]string{ts.Listener.Addr().String()})
	defer c.Close()
	resp, err := c.Do(NewRequest(strings.NewReader("ping")))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, _ := ioutil.ReadAll(resp.Body)
	expected := "req=1"
	if string(got) != expected {
		t.Errorf("Unexpected response for request #1; got %q; expected %q", string(got), expected)
	}

	t1 := time.Now()
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	buf := make([]byte, 1)
	n, err := conn.Read(buf)
	latency := time.Since(t1)
	if n != 0 || err != io.EOF {
		t.Errorf("Read = %v, %v, wanted %v, %v", n, err, 0, io.EOF)
	}
	if latency < 200*time.Millisecond {
		t.Errorf("got EOF after %s, want >= %s", latency, 200*time.Millisecond)
	}

	resp, err = c.Do(NewRequest(strings.NewReader("ping")))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, _ = ioutil.ReadAll(resp.Body)
	expected = "req=2"
	if string(got) != expected {
		t.Errorf("Unexpected response for request #2; got %q; expected %q", string(got), expected)
	}

}
*/

func TestClientWriteShutdown(t *testing.T) {
	defer afterTest(t)
	ts := nftest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {}))
	defer ts.Close()
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	err = conn.(*net.TCPConn).CloseWrite()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	donec := make(chan bool)
	go func() {
		defer close(donec)
		bs, err := ioutil.ReadAll(conn)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		got := string(bs)
		if got != "" {
			t.Errorf("read %q from server; want nothing", got)
		}
	}()
	select {
	case <-donec:
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout")
	}
}

func TestCloseNotifier(t *testing.T) {
	defer afterTest(t)
	gotReq := make(chan bool, 1)
	sawClose := make(chan bool, 1)
	ts := nftest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		gotReq <- true
		cc := w.(CloseNotifier).CloseNotify()
		<-cc
		sawClose <- true
	}))
	defer ts.Close()

	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	diec := make(chan bool)
	go func() {
		_, err := NewRequest(strings.NewReader("ping")).Write(conn)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		<-diec
		conn.Close()
	}()
For:
	for {
		select {
		case <-gotReq:
			diec <- true
		case <-sawClose:
			break For
		case <-time.After(5 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestCloseNotifierChanLeak(t *testing.T) {

}

type rwTestConn struct {
	io.Reader
	io.Writer
	noopConn

	closeFunc func() error // called if non-nil
	closec    chan bool    // else, if non-nil, send value to it on close
}

func (c *rwTestConn) Close() error {
	if c.closeFunc != nil {
		return c.closeFunc()
	}
	select {
	case c.closec <- true:
	default:
	}
	return nil
}

type noopConn struct{}

func (noopConn) LocalAddr() net.Addr                { return dummyAddr("local-addr") }
func (noopConn) RemoteAddr() net.Addr               { return dummyAddr("remote-addr") }
func (noopConn) SetDeadline(t time.Time) error      { return nil }
func (noopConn) SetReadDeadline(t time.Time) error  { return nil }
func (noopConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string {
	return string(a)
}

func (a dummyAddr) String() string {
	return string(a)
}

func TestAcceptMaxFds(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)
	ln := &errorListener{[]error{
		&net.OpError{
			Op:  "accept",
			Err: syscall.EMFILE,
		}}}
	err := Serve(ln, HandlerFunc(func(ResponseWriter, *Request) {}))
	if err != io.EOF {
		t.Errorf("got error %v, want EOF", err)
	}
}

type errorListener struct {
	errs []error
}

func (l *errorListener) Accept() (c net.Conn, err error) {
	if len(l.errs) == 0 {
		return nil, io.EOF
	}
	err = l.errs[0]
	l.errs = l.errs[1:]
	return
}

func (l *errorListener) Close() error {
	return nil
}

func (l *errorListener) Addr() net.Addr {
	return dummyAddr("test-address")
}

func BenchmarkClientServer(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	ts := nftest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		fmt.Fprintf(w, "Hello world.\n")
	}))
	defer ts.Close()
	b.StartTimer()

	c := NewClient([]string{ts.Listener.Addr().String()})
	defer c.Close()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(NewRequest(strings.NewReader("ping")))
		if err != nil {
			b.Fatalf("Do: %v", err)
		}
		all, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			b.Fatalf("ReadAll: %v", err)
		}
		body := string(all)
		if body != "Hello world.\n" {
			b.Fatalf("Got body: %v", body)
		}
	}

	b.StopTimer()
}

func benchmarkClientServerParallel(b *testing.B, parallelism int) {
	b.ReportAllocs()
	ts := nftest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		fmt.Fprintf(w, "Hello world.\n")
	}))
	defer ts.Close()
	c := NewClient([]string{ts.Listener.Addr().String()})
	defer c.Close()
	b.ResetTimer()
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := c.Do(NewRequest(strings.NewReader("ping")))
			if err != nil {
				b.Fatalf("Do: %v", err)
			}
			all, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				b.Fatalf("ReadAll: %v", err)
			}
			body := string(all)
			if body != "Hello world.\n" {
				b.Fatalf("Got body: %v", body)
			}
		}
	})
}

func BenchmarkClientServerParallel4(b *testing.B) {
	benchmarkClientServerParallel(b, 4)
}

func BenchmarkClientServerParallel64(b *testing.B) {
	benchmarkClientServerParallel(b, 64)
}
