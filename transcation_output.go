package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

// 交易的输出（UTXO）
type TxOutput struct {
	Value      uint64 // 未花费的金额
	PubKeyHash []byte // 持有人的公钥
}
type TxOutputs struct {
	Outputs []TxOutput
}

// Serialize 将 TxOutput 写入字节缓冲区
func (out *TxOutput) Serialize(buf *bytes.Buffer) error {
	// Value 8 字节小端
	if err := binary.Write(buf, binary.LittleEndian, out.Value); err != nil {
		return err
	}
	// PubKeyHash 变长
	WriteVarBytes(buf, out.PubKeyHash)
	return nil
}

// DeserializeTxOutput 从 io.Reader 读取 TxOutput
func DeserializeTxOutput(r io.Reader) (*TxOutput, error) {
	var value uint64
	if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
		return nil, err
	}
	pubKeyHash, err := ReadVarBytes(r)
	if err != nil {
		return nil, err
	}
	return &TxOutput{
		Value:      value,
		PubKeyHash: pubKeyHash,
	}, nil
}

// 修改 TxOutputs 的序列化/反序列化
func (outs TxOutputs) Serialize() []byte {
	buf := new(bytes.Buffer)
	WriteVarInt(buf, uint64(len(outs.Outputs)))
	for _, out := range outs.Outputs {
		_ = out.Serialize(buf)
	}
	return buf.Bytes()
}

func DeserializeOutputs(data []byte) TxOutputs {
	r := bytes.NewReader(data)
	count, err := ReadVarInt(r)
	if err != nil {
		log.Panic(err)
	}
	outputs := make([]TxOutput, 0, count)
	for i := uint64(0); i < count; i++ {
		out, err := DeserializeTxOutput(r)
		if err != nil {
			log.Panic(err)
		}
		outputs = append(outputs, *out)
	}
	return TxOutputs{Outputs: outputs}
}

func NewTxOutput(value uint64, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash, err := Base58Decode(address)
	if err != nil {
		return
	}
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

// 检查给定的公钥哈希是否与输出中存储的公钥哈希匹配
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}
