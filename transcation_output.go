package main

import (
	"bytes"
	"encoding/gob"
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

// 序列化
func (outs TxOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// 反序列化
func DeserializeOutputs(data []byte) TxOutputs {
	var outputs TxOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}

// 检查给定的公钥哈希是否与输出中存储的公钥哈希匹配
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}
