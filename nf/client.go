package nf

import (
	"bufio"
	"net"
	"time"
)

type Client struct {
	Timeout time.Duration
	net.Conn
	buf *bufio.ReadWriter
}

func Dial(addr string) (*Client, error) {
	c := new(Client)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c.Conn = conn
	br := newBufioReader(conn)
	bw := newBufioWriter(conn)
	c.buf = bufio.NewReadWriter(br, bw)
	return c, nil
}

func (c *Client) Do(req *Request) (resp *Response, err error) {
	err = c.Write(req)
	if err != nil {
		return nil, err
	}
	return ReadResponse(c.buf)
}

func (c *Client) Write(req *Request) error {
	_, err := req.Write(c.buf)
	if err != nil {
		return err
	}
	return c.buf.Flush()
}

func (c *Client) Close() {
	c.buf.Flush()
	c.Conn.Close()
	putBufioReader(c.buf.Reader)
	putBufioWriter(c.buf.Writer)
}
