package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"

	"gitlab.baidu.com/niushaofeng/gomcpack/mcpack"
)

type clientCodec struct {
	dec *Decoder
	enc *Encoder
	c   io.Closer

	req *Request

	resp *Response

	mutex   sync.Mutex
	pending map[uint64]string
}

func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		dec:     NewDecoder(conn),
		enc:     NewEncoder(conn),
		c:       conn,
		pending: make(map[uint64]string),
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mutex.Unlock()

	c.req = newClientRequest(r.ServiceMethod, param, r.Seq)
	content, _ := json.MarshalIndent(c.req, "", "  ")
	fmt.Println(string(content))
	return c.enc.Encode(&c.req.Head, &c.req.Body)
}

func newClientRequest(method string, param interface{}, id uint64) *Request {
	r := &Request{}
	r.Body.Content.Method = method
	r.Body.Content.Params = param
	r.Body.Content.Id = id
	return r
}

func newClientResponse() *Response {
	r := &Response{}
	r.Body.Content.Params = &Params{}
	return r
}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	c.resp = newClientResponse()
	if err := c.dec.Decode(&c.resp.Head, &c.resp.Body); err != nil {
		return err
	}
	header, content := c.resp.Body.Header, c.resp.Body.Content
	r.Error = ""
	r.Seq = content.Id
	if header.Errno != 0 {
		r.Error = header.ErrnoInfo
	}
	return nil
}

type resultParams struct {
	Params interface{} `json:"result_params"`
}

func (c *clientCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	params := c.resp.Body.Content.Params.(*Params)
	return mcpack.Unmarshal(params.Data, &resultParams{x})
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}

func NewClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn))
}

func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), err
}
