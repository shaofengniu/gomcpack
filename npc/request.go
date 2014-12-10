package npc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strings"
)

type Request struct {
	Header     Header
	Body       io.Reader
	RemoteAddr string
}

func (r *Request) Write(w io.Writer) (n int, err error) {
	n, err = r.Header.Write(w)
	if err != nil {
		return 0, err
	}
	written, err := io.Copy(w, r.Body)
	return int(written), err
}

func ReadRequest(r io.Reader) (req *Request, err error) {
	req = new(Request)
	_, err = req.Header.Read(r)
	if err != nil {
		return nil, err
	}
	if req.Header.MagicNum != HEADER_MAGICNUM {
		return nil, fmt.Errorf("invalid magic number %x", req.Header.MagicNum)
	}
	req.Body = io.LimitReader(r, int64(req.Header.BodyLen))
	return req, nil
}

func NewRequest(body io.Reader) *Request {
	req := new(Request)
	req.Header.LogId = rand.Uint32()
	req.Header.MagicNum = HEADER_MAGICNUM
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.Header.BodyLen = uint32(v.Len())
			req.Body = io.LimitReader(body, int64(req.Header.BodyLen))
		case *bytes.Reader:
			req.Header.BodyLen = uint32(v.Len())
			req.Body = io.LimitReader(body, int64(req.Header.BodyLen))
		case *strings.Reader:
			req.Header.BodyLen = uint32(v.Len())
			req.Body = io.LimitReader(body, int64(req.Header.BodyLen))
		default:
			panic("unsupported io.Reader")
		}
	}
	return req
}
