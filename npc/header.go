package npc

import (
	"encoding/binary"
	"errors"
	"io"
)

type Header struct {
	Id       uint16   `json:"id"`
	Version  uint16   `json:"version"`
	LogId    uint32   `json:"log_id"`
	Provider [16]byte `json:"provider"`
	MagicNum uint32   `json:"magic_num"`
	Reserved uint32   `json:"reserved"`
	BodyLen  uint32   `json:"bodylen"`
}

const HEADER_SIZE = 36

func (h *Header) Unmarshal(b []byte) error {
	if len(b) < HEADER_SIZE {
		return errors.New("incomplete header")
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
		return errors.New("not enough buffer for header")
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

func (h *Header) Write(w io.Writer) (n int, err error) {
	var buf [HEADER_SIZE]byte
	if err = h.Marshal(buf[:]); err != nil {
		return 0, err
	}
	return w.Write(buf[:])
}

func (h *Header) Read(r io.Reader) (n int, err error) {
	var buf [HEADER_SIZE]byte
	if n, err = io.ReadFull(r, buf[:]); err != nil {
		return n, err
	}
	if err = h.Unmarshal(buf[:]); err != nil {
		return n, err
	}
	return n, nil
}
