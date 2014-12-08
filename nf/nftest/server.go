package nftest

import (
	"fmt"
	"net"
	"sync"

	"gitlab.baidu.com/niushaofeng/gomcpack/nf"
)

type Server struct {
	Listener net.Listener
	Config   *nf.Server

	// wg counts the number of outstanding requests on this server
	// Close blocks until all requests are finished
	wg sync.WaitGroup
}

// historyListener keeps track of all the connections that it's ever
// accepted
type historyListener struct {
	net.Listener
	sync.Mutex
	history []net.Conn
}

func (hs *historyListener) Accept() (c net.Conn, err error) {
	c, err = hs.Listener.Accept()
	if err == nil {
		hs.Lock()
		hs.history = append(hs.history, c)
		hs.Unlock()
	}
	return
}

func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("nftest: failed to listen on port: %v", err))
	}
	return l
}

// NewServer starts and returns a new server.
// The caller should call Close when finished to shut it down
func NewServer(handler nf.Handler) *Server {
	ts := NewUnstartedServer(handler)
	ts.Start()
	return ts
}

func NewUnstartedServer(handler nf.Handler) *Server {
	return &Server{
		Listener: newLocalListener(),
		Config:   &nf.Server{Handler: handler},
	}
}

func (s *Server) Start() {
	s.Listener = &historyListener{Listener: s.Listener}
	s.wrapHandler()
	go s.Config.Serve(s.Listener)
}

func (s *Server) wrapHandler() {
	s.Config.Handler = &waitGroupHandler{
		s: s,
		h: s.Config.Handler,
	}
}

// Close shuts down the server and blocks until all outstanding
// requests on this server have completed
func (s *Server) Close() {
	s.Listener.Close()
	s.wg.Wait()
	s.CloseClientConnections()
}

// CloseClientConnections closes any currently open connections to the
// test server
func (s *Server) CloseClientConnections() {
	hl, ok := s.Listener.(*historyListener)
	if !ok {
		return
	}
	hl.Lock()
	for _, conn := range hl.history {
		conn.Close()
	}
	hl.Unlock()
}

type waitGroupHandler struct {
	s *Server
	h nf.Handler
}

func (h *waitGroupHandler) Serve(w nf.ResponseWriter, r *nf.Request) {
	h.s.wg.Add(1)
	defer h.s.wg.Done()
	h.h.Serve(w, r)
}
