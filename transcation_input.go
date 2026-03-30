package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// 当前交易的输入（拥有者）
type TxInput struct {
	Txid      []byte // 上一个交易的id
	Vout      uint64 // 上一个交易的输出索引
	Signature []byte // 上一个交易的签名
	PubKey    []byte // 公钥（用于验证签名）
}

// Serialize 将 TxInput 写入字节缓冲区
func (in *TxInput) Serialize(buf *bytes.Buffer) error {
	// Txid 固定 32 字节
	if len(in.Txid) != 32 {
		return fmt.Errorf("invalid txid length")
	}
	buf.Write(in.Txid)
	// Vout 4 字节小端
	if err := binary.Write(buf, binary.LittleEndian, in.Vout); err != nil {
		return err
	}
	// Signature 变长
	WriteVarBytes(buf, in.Signature)
	// PubKey 变长
	WriteVarBytes(buf, in.PubKey)
	return nil
}

// DeserializeTxInput 从 io.Reader 读取 TxInput
func DeserializeTxInput(r io.Reader) (*TxInput, error) {
	txid := make([]byte, 32)
	if _, err := io.ReadFull(r, txid); err != nil {
		return nil, err
	}
	var vout uint64
	if err := binary.Read(r, binary.LittleEndian, &vout); err != nil {
		return nil, err
	}
	sig, err := ReadVarBytes(r)
	if err != nil {
		return nil, err
	}
	pubKey, err := ReadVarBytes(r)
	if err != nil {
		return nil, err
	}
	return &TxInput{
		Txid:      txid,
		Vout:      vout,
		Signature: sig,
		PubKey:    pubKey,
	}, nil
}
