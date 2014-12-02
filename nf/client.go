package nf

import (
	"bufio"
	"net"
	"time"
)

type Client struct {
	Timeout time.Duration
	rwc     net.Conn
	buf     *bufio.ReadWriter
}

func Dial(addr string) (*Client, error) {
	c := new(Client)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c.rwc = conn
	br := newBufioReader(conn)
	bw := newBufioWriter(conn)
	c.buf = bufio.NewReadWriter(br, bw)
	return c, nil
}

func (c *Client) Do(req *Request) (resp *Response, err error) {
	_, err = req.Write(c.buf)
	if err != nil {
		return nil, err
	}
	err = c.buf.Flush()
	if err != nil {
		return nil, err
	}
	return ReadResponse(c.buf)
}
