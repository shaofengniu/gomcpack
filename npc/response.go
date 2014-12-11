package npc

import (
	"fmt"
	"io"
)

type Response struct {
	Header Header
	Body   []byte
}

func ReadResponse(r io.Reader) (resp *Response, err error) {
	resp = new(Response)
	_, err = resp.Header.Read(r)
	if err != nil {
		return nil, err
	}
	if resp.Header.MagicNum != HEADER_MAGICNUM {
		return nil, fmt.Errorf("invalid magic number %x", resp.Header.MagicNum)
	}

	resp.Body = make([]byte, int(resp.Header.BodyLen))
	_, err = io.ReadFull(r, resp.Body)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
