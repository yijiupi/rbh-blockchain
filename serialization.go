package main

import (
	"bytes"
	"encoding/binary"
	"io"
)

// WriteVarInt 将 uint64 写入 bytes.Buffer 使用比特币 varint 编码
func WriteVarInt(buf *bytes.Buffer, val uint64) {
	if val < 0xfd {
		buf.WriteByte(byte(val))
	} else if val <= 0xffff {
		buf.WriteByte(0xfd)
		_ = binary.Write(buf, binary.LittleEndian, uint16(val))
	} else if val <= 0xffffffff {
		buf.WriteByte(0xfe)
		_ = binary.Write(buf, binary.LittleEndian, uint32(val))
	} else {
		buf.WriteByte(0xff)
		_ = binary.Write(buf, binary.LittleEndian, val)
	}
}

// ReadVarInt 从 io.Reader 读取比特币 varint
func ReadVarInt(r io.Reader) (uint64, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	if err != nil {
		return 0, err
	}
	switch b[0] {
	case 0xfd:
		var val uint16
		err = binary.Read(r, binary.LittleEndian, &val)
		return uint64(val), err
	case 0xfe:
		var val uint32
		err = binary.Read(r, binary.LittleEndian, &val)
		return uint64(val), err
	case 0xff:
		var val uint64
		err = binary.Read(r, binary.LittleEndian, &val)
		return val, err
	default:
		return uint64(b[0]), nil
	}
}

// WriteVarBytes 写入变长字节数组（长度+内容）
func WriteVarBytes(buf *bytes.Buffer, data []byte) {
	WriteVarInt(buf, uint64(len(data)))
	buf.Write(data)
}

// ReadVarBytes 读取变长字节数组
func ReadVarBytes(r io.Reader) ([]byte, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	data := make([]byte, length)
	_, err = io.ReadFull(r, data)
	return data, err
}
