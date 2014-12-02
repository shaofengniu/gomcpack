package nf

import (
	"bufio"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

type conn struct {
	remoteAddr   string   // network address of remote side
	server       *Server  // the Server on which the connection arrived
	rwc          net.Conn // i/o connection
	buf          *bufio.ReadWriter
	mu           sync.Mutex // guards the following
	clientGone   bool       // if client has disconnected mid-request
	closeNotifyc chan bool  // made lazily
}

var debugServerConnections = false

func (srv *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = new(conn)
	c.remoteAddr = rwc.RemoteAddr().String()
	c.server = srv
	c.rwc = rwc
	if debugServerConnections {
		c.rwc = newLoggingConn("server", c.rwc)
	}
	br := newBufioReader(c.rwc)
	bw := newBufioWriter(c.rwc)
	c.buf = bufio.NewReadWriter(br, bw)
	return c, nil
}

func (c *conn) serve() {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("nf: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
	}()

	for {
		w, err := c.readRequest()
		if err != nil {
			if err == io.EOF {
				break
			} else if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				break
			}
			// TODO: reply bad request
			break
		}
		c.setState(c.rwc, StateActive)
		serveHandler{c.server}.Serve(w, w.req)
		w.finishRequest()
		c.setState(c.rwc, StateIdle)
	}
}

func (c *conn) readRequest() (w *response, err error) {
	if d := c.server.ReadTimeout; d != 0 {
		c.rwc.SetReadDeadline(time.Now().Add(d))
	}
	if d := c.server.WriteTimeout; d != 0 {
		defer func() {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}()
	}
	var req *Request
	if req, err = ReadRequest(c.buf); err != nil {
		return nil, err
	}
	req.RemoteAddr = c.remoteAddr

	w = &response{
		conn:          c,
		req:           req,
		handlerHeader: req.Header,
	}
	w.handlerHeader.BodyLen = 0
	return w, nil
}

func (c *conn) setState(nc net.Conn, state ConnState) {

}

type ResponseWriter interface {
	Header() *Header
	Write([]byte) (int, error)
}

type response struct {
	conn        *conn
	req         *Request
	wroteHeader bool // reply header has been written

	handlerHeader Header
}

func (w *response) Header() *Header {
	return &w.handlerHeader
}

func (w *response) Write(data []byte) (n int, err error) {
	if len(data) == 0 {
		return 0, nil
	}
	w.handlerHeader.BodyLen = uint32(len(data))
	n, err = w.handlerHeader.Write(w.conn.buf)
	if err != nil {
		return 0, nil
	}
	return w.conn.buf.Write(data)
}

func (w *response) finishRequest() {
	w.conn.buf.Flush()
}

type Server struct {
	Addr         string        // TCP address to listen on
	Handler      Handler       // handler to invoke
	ReadTimeout  time.Duration // maximum duration before timing out read of the request
	WriteTimeout time.Duration // maximum duration before timing out write of the response
	ErrorLog     *log.Logger   // If nil, logging goes to os.Stderr via the log package's standard logger

}

type ConnState int

const (
	StateNew ConnState = iota
	StateActive
	StateIdle
	StateClosed
)

var stateName = map[ConnState]string{
	StateNew:    "new",
	StateActive: "active",
	StateIdle:   "idle",
	StateClosed: "closed",
}

func (c ConnState) String() string {
	return stateName[c]
}

type serveHandler struct {
	srv *Server
}

func (sh serveHandler) Serve(rw ResponseWriter, req *Request) {
	sh.srv.Handler.Serve(rw, req)
}

func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":8888"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("nf: accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c, err := srv.newConn(rw)
		if err != nil {
			continue
		}
		c.setState(c.rwc, StateNew)
		go c.serve()
	}
}

func (srv *Server) logf(format string, args ...interface{}) {
	if srv.ErrorLog != nil {
		srv.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

var (
	bufioReaderPool sync.Pool
	bufioWriterPool sync.Pool
)

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	return bufio.NewReader(r)
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

func newBufioWriter(w io.Writer) *bufio.Writer {
	if v := bufioWriterPool.Get(); v != nil {
		bw := v.(*bufio.Writer)
		bw.Reset(w)
		return bw
	}
	return bufio.NewWriter(w)
}

func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	bufioWriterPool.Put(bw)
}

func newLoggingConn(baseName string, c net.Conn) net.Conn {
	// TODO
	return c
}

type Handler interface {
	Serve(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Serve(w ResponseWriter, r *Request) {
	f(w, r)
}
