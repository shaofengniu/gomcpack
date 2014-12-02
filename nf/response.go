package nf

import (
	"io"
)

type Response struct {
	Header Header
	Body   io.Reader
}

func ReadResponse(r io.Reader) (resp *Response, err error) {
	resp = new(Response)
	_, err = resp.Header.Read(r)
	if err != nil {
		return nil, err
	}
	resp.Body = io.LimitReader(r, int64(resp.Header.BodyLen))
	return resp, nil
}
