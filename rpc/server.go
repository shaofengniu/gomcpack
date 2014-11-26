package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"sync"

	"gitlab.baidu.com/niushaofeng/gomcpack/mcpack"
)

type serverCodec struct {
	dec *Decoder
	enc *Encoder
	c   io.Closer

	req *Request

	mutex   sync.Mutex
	seq     uint64
	pending map[uint64]*Request
}

type Request struct {
	Head Head        `json:"head"`
	Body RequestBody `json:"body"`
}

func newServerRequest() *Request {
	r := &Request{}
	r.Body.Content.Params = &Params{}
	return r
}

type RequestBody struct {
	Header  RequestHeader  `json:"header"`
	Content RequestContent `json:"content"`
}

type RequestHeader struct {
}

type RequestContent struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	Id     uint64      `json:"id"`
}

type Response struct {
	Head Head         `json:"head"`
	Body ResponseBody `json:"body"`
}

func newServerResponse(req *Request, x interface{}) *Response {
	r := &Response{}
	r.Body.Content.Params = x
	return r
}

type ResponseBody struct {
	Header  ResponseHeader  `json:"header"`
	Content ResponseContent `json:"content"`
}

type ResponseHeader struct {
	Errno     uint32 `json:"err_no"`
	ErrnoInfo string `json:"errno_info"`
}

type ResponseContent struct {
	Id     uint64      `json:"id"`
	Params interface{} `json:"result_params"`
}

type Params struct {
	Data []byte
}

func (p *Params) UnmarshalMCPACK(b []byte) error {
	p.Data = b
	return nil
}

func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		dec:     NewDecoder(conn),
		enc:     NewEncoder(conn),
		c:       conn,
		pending: make(map[uint64]*Request),
	}
}

func (c *serverCodec) ReadRequestHeader(r *rpc.Request) error {
	c.req = newServerRequest()
	if err := c.dec.Decode(&c.req.Head, &c.req.Body); err != nil {
		return err
	}
	cc, _ := json.MarshalIndent(c.req, "", "  ")
	fmt.Println(string(cc))
	content := c.req.Body.Content
	r.ServiceMethod = content.Method

	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = c.req
	r.Seq = c.seq
	c.mutex.Unlock()

	return nil
}

type requestParams struct {
	Params interface{} `json:"params"`
}

func (c *serverCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}
	params := c.req.Body.Content.Params.(*Params)
	return mcpack.Unmarshal(params.Data, &requestParams{x})
}

func (c *serverCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	c.mutex.Lock()
	req, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	resp := newServerResponse(req, x)
	return c.enc.Encode(&resp.Head, &resp.Body)
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}

func ServeConn(conn io.ReadWriteCloser) {
	rpc.ServeCodec(NewServerCodec(conn))
}
