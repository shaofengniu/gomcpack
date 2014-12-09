package npc

import (
	"bufio"
	"net"
	"sync"
	"time"
)

const (
	// DefaultTimeout is the default socket read/write timeout.
	DefaultTimeout = 100 * time.Millisecond

	MaxIdleConnsPerAddr = 2
)

type Client struct {
	Timeout time.Duration

	selector ServerSelector

	sync.Mutex
	freeconn map[string][]*clientConn
}

func (c *Client) putFreeConn(addr net.Addr, cn *clientConn) {
	c.Lock()
	defer c.Unlock()
	if c.freeconn == nil {
		c.freeconn = make(map[string][]*clientConn)
	}
	freelist := c.freeconn[addr.String()]
	if len(freelist) >= MaxIdleConnsPerAddr {
		cn.close()
		return
	}
	c.freeconn[addr.String()] = append(freelist, cn)
}

func (c *Client) getFreeConn(addr net.Addr) (cn *clientConn, ok bool) {
	c.Lock()
	defer c.Unlock()
	if c.freeconn == nil {
		return nil, false
	}
	freelist, ok := c.freeconn[addr.String()]
	if !ok || len(freelist) == 0 {
		return nil, false
	}
	cn = freelist[len(freelist)-1]
	c.freeconn[addr.String()] = freelist[:len(freelist)-1]
	return cn, true
}

func (c *Client) netTimeout() time.Duration {
	if c.Timeout != 0 {
		return c.Timeout
	}
	return DefaultTimeout
}

type clientConn struct {
	nc   net.Conn
	rw   *bufio.ReadWriter
	addr net.Addr
	c    *Client
}

func (cn *clientConn) close() error {
	return cn.nc.Close()
}

func (cn *clientConn) release() {
	cn.c.putFreeConn(cn.addr, cn)
}

func (cn *clientConn) extendDeadline() {
	cn.nc.SetDeadline(time.Now().Add(cn.c.netTimeout()))
}

// condRelease releases this connection if the error pointed by err is
// nil or is only a protocol level error. The purpose is to not
// recycle TCP connections that are bad
func (cn *clientConn) condRelease(err *error) {
	if *err == nil || resumableError(*err) {
		cn.release()
	} else {
		cn.close()
	}
}

func resumableError(err error) bool {
	return false
}

func NewClient(server []string) *Client {
	ss := new(ServerList)
	ss.SetServers(server)
	return NewFromSelector(ss)
}

func NewFromSelector(ss ServerSelector) *Client {
	return &Client{selector: ss}
}

func (c *Client) Do(req *Request) (resp *Response, err error) {
	err = c.withConn(func(rw *bufio.ReadWriter) error {
		if _, err := req.Write(rw); err != nil {
			return err
		}
		if err := rw.Flush(); err != nil {
			return err
		}
		rsp, err := ReadResponse(rw)
		if err != nil {
			return err
		}
		resp = rsp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) withConn(fn func(*bufio.ReadWriter) error) error {
	addr, err := c.selector.PickServer()
	if err != nil {
		return err
	}
	cn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	defer cn.condRelease(&err)
	return fn(cn.rw)
}

func (c *Client) getConn(addr net.Addr) (cn *clientConn, err error) {
	cn, ok := c.getFreeConn(addr)
	if ok {
		cn.extendDeadline()
		return cn, nil
	}
	nc, err := c.dial(addr)
	if err != nil {
		return nil, err
	}
	cn = &clientConn{
		nc:   nc,
		addr: addr,
		rw:   bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
		c:    c,
	}
	cn.extendDeadline()
	return cn, nil
}

func (c *Client) dial(addr net.Addr) (net.Conn, error) {
	nc, err := net.DialTimeout(addr.Network(), addr.String(), c.netTimeout())
	if err == nil {
		return nc, nil
	}
	return nil, err
}

func (c *Client) Close() error {
	c.Lock()
	defer c.Unlock()
	for _, freelist := range c.freeconn {
		for _, clientConn := range freelist {
			clientConn.close()
		}
	}
	return nil
}
