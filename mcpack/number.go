package mcpack

import (
	"encoding/binary"
	"math"
)

/*func Int8(b []byte) int8 {
	return int8(b[0])
}

func PutInt8(b []byte, v int8) {
	b[0] = byte(uint8(v))
}

func Int16(b []byte) int16 {
	return int16(uint16(b[0]) + uint16(b[1])<<8)
}

func PutInt16(b []byte, v int16) {
	b[0] = byte(uint16(v))
	b[1] = byte(uint16(v) >> 8)
}*/

func Int32(b []byte) int32 {
	return int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}

func PutInt32(b []byte, v int32) {
	b[0] = byte(uint32(v))
	b[1] = byte(uint32(v) >> 8)
	b[2] = byte(uint32(v) >> 16)
	b[3] = byte(uint32(v) >> 24)
}

func Int64(b []byte) int64 {
	return int64(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
}

func PutInt64(b []byte, v int64) {
	b[0] = byte(uint64(v))
	b[1] = byte(uint64(v) >> 8)
	b[2] = byte(uint64(v) >> 16)
	b[3] = byte(uint64(v) >> 24)
	b[4] = byte(uint64(v) >> 32)
	b[5] = byte(uint64(v) >> 40)
	b[6] = byte(uint64(v) >> 48)
	b[7] = byte(uint64(v) >> 56)
}

func Uint8(b []byte) uint8 {
	return uint8(b[0])
}

func PutUint8(b []byte, v uint8) {
	b[0] = byte(v)
}

func Uint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func PutUint16(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func Uint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func PutUint32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func Uint64(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func PutUint64(b []byte, v uint64) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
}

func Float32(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	return math.Float32frombits(bits)
}

func PutFloat32(b []byte, v float32) {
	bits := math.Float32bits(v)
	binary.LittleEndian.PutUint32(b, bits)
}

func Float64(b []byte) float64 {
	bits := binary.LittleEndian.Uint64(b)
	return math.Float64frombits(bits)
}

func PutFloat64(b []byte, v float64) {
	bits := math.Float64bits(v)
	binary.LittleEndian.PutUint64(b, bits)
}
