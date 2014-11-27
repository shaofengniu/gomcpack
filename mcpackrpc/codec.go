package mcpackrpc

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.baidu.com/niushaofeng/gomcpack/mcpack"
)

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type Encoder struct {
	w   io.Writer
	buf [HEAD_SIZE]byte
}

func (e *Encoder) Encode(h *Head, m interface{}) error {
	bytes, err := mcpack.Marshal(m)
	if err != nil {
		return err
	}
	h.BodyLen = uint32(len(bytes))
	err = h.Marshal(e.buf[:])
	if err != nil {
		return err
	}
	_, err = e.w.Write(e.buf[:])
	if err != nil {
		return err
	}
	_, err = e.w.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

type Decoder struct {
	r   io.Reader
	buf []byte
	err error
}

func (d *Decoder) resizeIfNeeded(n int) {
	if cap(d.buf)-len(d.buf) < n {
		newbuf := make([]byte, len(d.buf), 2*cap(d.buf)+n)
		copy(newbuf, d.buf)
		d.buf = newbuf
	}
}

func (d *Decoder) reset() {
	d.buf = d.buf[:0]
}

func (d *Decoder) Decode(h *Head, m interface{}) error {
	if d.err != nil {
		return d.err
	}
	d.reset()
	d.resizeIfNeeded(HEAD_SIZE)
	_, err := d.r.Read(d.buf[:HEAD_SIZE])
	if err != nil {
		return err
	}
	d.buf = d.buf[:HEAD_SIZE]
	err = h.Unmarshal(d.buf)
	if err != nil {
		return err
	}
	d.resizeIfNeeded(int(h.BodyLen))
	_, err = d.r.Read(d.buf[HEAD_SIZE : HEAD_SIZE+h.BodyLen])
	if err != nil {
		return err
	}
	d.buf = d.buf[:HEAD_SIZE+h.BodyLen]
	err = mcpack.Unmarshal(d.buf[HEAD_SIZE:HEAD_SIZE+h.BodyLen], m)
	if err != nil {
		return err
	}
	return nil
}

type Head struct {
	Id       uint16   `json:"id"`
	Version  uint16   `json:"version"`
	LogId    uint32   `json:"log_id"`
	Provider [16]byte `json:"provider"`
	MagicNum uint32   `json:"magic_num"`
	Reserved uint32   `json:"reserved"`
	BodyLen  uint32   `json:"bodylen"`
}

const HEAD_SIZE = 36

func (n *Head) Unmarshal(b []byte) error {
	if len(b) < HEAD_SIZE {
		return errors.New("incomplete head")
	}
	n.Id = binary.LittleEndian.Uint16(b[0:2])
	n.Version = binary.LittleEndian.Uint16(b[2:4])
	n.LogId = binary.LittleEndian.Uint32(b[4:8])
	copy(n.Provider[:], b[8:24])
	n.MagicNum = binary.LittleEndian.Uint32(b[24:28])
	n.Reserved = binary.LittleEndian.Uint32(b[28:32])
	n.BodyLen = binary.LittleEndian.Uint32(b[32:36])
	return nil
}

func (n *Head) Marshal(b []byte) error {
	if len(b) < HEAD_SIZE {
		return errors.New("not enough buffer")
	}
	binary.LittleEndian.PutUint16(b[0:2], n.Id)
	binary.LittleEndian.PutUint16(b[2:4], n.Version)
	binary.LittleEndian.PutUint32(b[4:8], n.LogId)
	copy(b[8:24], n.Provider[:])
	binary.LittleEndian.PutUint32(b[24:28], n.MagicNum)
	binary.LittleEndian.PutUint32(b[28:32], n.Reserved)
	binary.LittleEndian.PutUint32(b[32:36], n.BodyLen)
	return nil
}
