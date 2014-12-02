package nf

import (
	"encoding/binary"
	"io"
)

type Request struct {
	Header Header
	Body   io.LimitedReader
}

func ReadRequest(b *bufio.Reader) (req *Request, err error) {
	req := new(Request)

	var buf [HEADER_SIZE]byte
	_, err := io.ReadFull(b, buf[:])
	if err != nil {
		return nil, err
	}

	err = req.Header.Unmarshal(b[:])
	if err != nil {
		return nil, err
	}

	req.Body = io.LimitReader(b, req.Header.BodyLen)
	return req, nil
}

type Header struct {
	Id       uint16   `json:"id"`
	Version  uint16   `json:"version"`
	LogId    uint16   `json:"log_id"`
	Provider [16]byte `json:"provider"`
	MagicNum uint32   `json:"magic_num"`
	Reserved uint32   `json:"reserved"`
	BodyLen  uint32   `json:"bodylen"`
}

const HEADER_SIZE = 36

func (h *Header) Unmarshal(b []byte) error {
	if len(b) < HEADER_SIZE {
		return errors.New("incomplete head")
	}
	h.Id = binary.LittleEndian.Uint16(b[0:2])
	h.Version = binary.LittleEndian.Uint16(b[2:4])
	h.LogId = binary.LittleEndian.Uint32(b[4:8])
	copy(h.Provider[:], b[8:24])
	h.MagicNum = binary.LittleEndian.Uint32(b[24:28])
	h.Reserved = binary.LittleEndian.Uint32(b[28:32])
	h.BodyLen = binary.LittleEndian.Uint32(b[32:36])
	return nil
}

func (h *Header) Marshal(b []byte) error {
	if len(b) < HEADER_SIZE {
		return errors.New("not enough buffer")
	}
	binary.LittleEndian.PutUint16(b[0:2], h.Id)
	binary.LittleEndian.PutUint16(b[2:4], h.Version)
	binary.LittleEndian.PutUint32(b[4:8], h.LogId)
	copy(b[8:24], h.Provider[:])
	binary.LittleEndian.PutUint32(b[24:28], h.MagicNum)
	binary.LittleEndian.PutUint32(b[28:32], h.Reserved)
	binary.LittleEndian.PutUint32(b[32:36], h.BodyLen)
	return nil
}
