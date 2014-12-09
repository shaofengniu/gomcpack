package npc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

// Errors
var (
	ErrWroteResponse = errors.New("response has already been written")
)

type CloseNotifier interface {
	CloseNotify() <-chan struct{}
}

type conn struct {
	remoteAddr   string   // network address of remote side
	server       *Server  // the Server on which the connection arrived
	rwc          net.Conn // i/o connection
	sr           liveSwitchReader
	buf          *bufio.ReadWriter // buffered reader/writer for rwc
	mu           sync.Mutex        // guards the following
	clientGone   bool              // if client has disconnected mid-request
	closeNotifyc chan struct{}     // made lazily
}

// debugServerConnections controls whether all server connections are
// wrapped with a verbose logging wrapper
var debugServerConnections = false

// Create new connection from rwc
func (srv *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = new(conn)
	c.remoteAddr = rwc.RemoteAddr().String()
	c.server = srv
	c.rwc = rwc
	if debugServerConnections {
		c.rwc = newLoggingConn("server", c.rwc)
	}
	c.sr = liveSwitchReader{r: c.rwc}
	br := newBufioReader(&c.sr)
	bw := newBufioWriter(c.rwc)
	c.buf = bufio.NewReadWriter(br, bw)
	return c, nil
}

func (c *conn) serve() {
	origConn := c.rwc
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("nf: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}

		c.close()
		c.setState(origConn, StateClosed)
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

func (c *conn) finalFlush() {
	if c.buf != nil {
		c.buf.Flush()
		putBufioReader(c.buf.Reader)
		putBufioWriter(c.buf.Writer)
		c.buf = nil
	}
}

func (c *conn) close() {
	c.finalFlush()
	if c.rwc != nil {
		c.rwc.Close()
		c.rwc = nil
	}
}

func (c *conn) closeNotify() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closeNotifyc == nil {
		c.closeNotifyc = make(chan struct{}, 1)
		pr, pw := io.Pipe()
		readSource := c.sr.Swap(pr)
		go func() {
			_, err := io.Copy(pw, readSource)
			if err == nil {
				err = io.EOF
			}
			pw.CloseWithError(err)
			c.noteClientGone()
		}()
	}
	return c.closeNotifyc
}

func (c *conn) noteClientGone() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closeNotifyc != nil && !c.clientGone {
		close(c.closeNotifyc)
	}
	c.clientGone = true
}

// A liveSwitchReader can have its Reader changed at runtime.
type liveSwitchReader struct {
	sync.Mutex
	r io.Reader
}

func (sr *liveSwitchReader) Read(p []byte) (n int, err error) {
	sr.Lock()
	r := sr.r
	sr.Unlock()
	return r.Read(p)
}

func (sr *liveSwitchReader) Swap(i io.Reader) (o io.Reader) {
	sr.Lock()
	o = sr.r
	sr.r = i
	sr.Unlock()
	return o
}

type ResponseWriter interface {
	Header() *Header
	Write([]byte) (int, error)
}

type response struct {
	conn          *conn
	req           *Request
	wroteResponse bool // reply has already been written

	handlerHeader Header
}

func (w *response) Header() *Header {
	return &w.handlerHeader
}

func (w *response) Write(data []byte) (n int, err error) {
	if w.wroteResponse {
		return 0, ErrWroteResponse
	}
	w.wroteResponse = true
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

func (w *response) CloseNotify() <-chan struct{} {
	return w.conn.closeNotify()
}

func Serve(l net.Listener, handler Handler) error {
	srv := &Server{Handler: handler}
	return srv.Serve(l)
}

type Server struct {
	Addr         string        // TCP address to listen on
	Handler      Handler       // handler to invoke
	ReadTimeout  time.Duration // maximum duration before timing out read of the request
	WriteTimeout time.Duration // maximum duration before timing out write of the response
	ErrorLog     *log.Logger   // If nil, logging goes to os.Stderr via the log package's standard logger

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state.
	ConnState func(net.Conn, ConnState)
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

var (
	uniqNameMu   sync.Mutex
	uniqNameNext = make(map[string]int)
)

func newLoggingConn(baseName string, c net.Conn) net.Conn {
	uniqNameMu.Lock()
	defer uniqNameMu.Unlock()
	uniqNameNext[baseName]++
	return &loggingConn{
		name: fmt.Sprintf("%s-%d", baseName, uniqNameNext[baseName]),
		Conn: c,
	}
}

type loggingConn struct {
	name string
	net.Conn
}

func (c *loggingConn) Write(p []byte) (n int, err error) {
	log.Printf("%s.Write(%d) = ....", c.name, len(p))
	n, err = c.Conn.Write(p)
	log.Printf("%s.Write(%d) = %d, %v", c.name, len(p), n, err)
	return
}

func (c *loggingConn) Read(p []byte) (n int, err error) {
	log.Printf("%s.Read(%d) = ....", c.name, len(p))
	n, err = c.Conn.Read(p)
	log.Printf("%s.Read(%d) = %d, %v", c.name, len(p), n, err)
	return
}

func (c *loggingConn) Close() (err error) {
	log.Printf("%s.Close() = ...", c.name)
	err = c.Conn.Close()
	log.Printf("%s.Close() = %v", c.name, err)
	return
}

// Objects implementing the Handler interface can be registered to
// serve client requests
type Handler interface {
	Serve(ResponseWriter, *Request)
}

// The HandlerFunc type is an adapter to allow the use of ordinary
// functions as handlers.
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Serve(w ResponseWriter, r *Request) {
	f(w, r)
}
